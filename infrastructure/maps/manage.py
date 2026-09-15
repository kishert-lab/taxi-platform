#!/usr/bin/env python3
"""Isolated OSM builds and explicit release activation. Linux, Python 3.10+, Docker Compose."""
import argparse
from contextlib import contextmanager
from datetime import datetime, timezone
import fcntl
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
import time
import urllib.parse
import urllib.request
import zipfile
from assets import write_assets, audit_style
from vector_tile import layer_counts

HERE = Path(__file__).resolve().parent
REPOSITORY = HERE.parent.parent
VERSION = re.compile(r"^[a-z0-9][a-z0-9_-]{0,55}$")

def read_json(path):
    return json.loads(Path(path).read_text(encoding="utf-8-sig"))

def atomic_json(path, value):
    path = Path(path)
    staging = path.with_suffix(path.suffix + ".new")
    staging.write_text(json.dumps(value,ensure_ascii=False,indent=2), encoding="utf-8")
    os.replace(staging,path)

def now():
    return datetime.now(timezone.utc).isoformat()

class Pipeline:
    def __init__(self, config):
        self.config = config
        self.root = Path(config["root"]).resolve()
        if self.root == Path("/") or not self.root.is_absolute():
            raise ValueError("dedicated absolute map root required")
        self.root.mkdir(parents=True, exist_ok=True)
        (self.root/"releases").mkdir(exist_ok=True)
        self.images = read_json(HERE/"images.lock.json")
        self.log = None

    @contextmanager
    def lock(self):
        with (self.root/"update.lock").open("w") as lock:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
            yield

    def release(self, version):
        if not VERSION.fullmatch(version): raise ValueError("invalid release version")
        target = self.root/"releases"/version
        if target.is_symlink() or target.resolve().parent != (self.root/"releases").resolve():
            raise ValueError("release path escapes map root")
        return target

    def run(self, args):
        print(json.dumps({"operation":"maps.command","command":args,"at":now()}),flush=True)
        subprocess.run(args,check=True,cwd=REPOSITORY,stdout=self.log,stderr=subprocess.STDOUT)

    def inspect(self, name):
        return json.loads(subprocess.check_output(["docker","inspect",name]))[0]

    def address(self, version, service, port):
        info = self.inspect(f"maps-{version}-{service}")
        address = info["NetworkSettings"]["Networks"][self.config["network"]]["IPAddress"]
        return f"http://{address}:{port}"

    def http(self, url):
        with urllib.request.urlopen(url,timeout=15) as response:
            return json.load(response)

    def wait_http(self,url,predicate,attempts=90):
        last = None
        for _ in range(attempts):
            try:
                value=self.http(url)
                if predicate(value): return value
                last=ValueError("readiness response failed validation")
            except (OSError,ValueError) as error:
                last=error
            time.sleep(2)
        raise RuntimeError(f"readiness failed for {url}: {last}")

    def network(self):
        result=subprocess.run(["docker","network","inspect",self.config["network"]],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
        if result.returncode: self.run(["docker","network","create",self.config["network"]])

    def compose(self,release,*args):
        self.run(["docker","compose","-p","maps-"+release.name,"-f",str(release/"compose.json"),*args])

    def generate_compose(self,release):
        version=release.name
        services={}
        for name,image,command in [("osrm",self.images["osrm"],["osrm-routed","--algorithm","mld","/data/region.osrm"]),("es",self.images["elasticsearch"],None),("pelias",self.images["pelias_api"],None),("libpostal",self.images["libpostal"],None)]:
            entry={"image":image,"container_name":f"maps-{version}-{name}","restart":"unless-stopped","networks":["maps"]}
            if command: entry["command"]=command
            services[name]=entry
        services["osrm"].update({"volumes":[str(release)+":/data:ro"],"mem_limit":self.config["build_memory"],"cpus":self.config["build_cpus"]})
        services["es"].update({"environment":{"discovery.type":"single-node","xpack.security.enabled":"false","ES_JAVA_OPTS":"-Xms"+self.config["es_heap"]+" -Xmx"+self.config["es_heap"]},"volumes":[str(release/"elasticsearch")+":/usr/share/elasticsearch/data"],"healthcheck":{"test":["CMD-SHELL","curl -fsS 'http://localhost:9200/_cluster/health?wait_for_status=yellow' >/dev/null"],"interval":"10s","timeout":"5s","retries":30}})
        services["pelias"].update({"environment":{"PELIAS_CONFIG":"/code/pelias.json","PORT":"4000","HOST":"0.0.0.0"},"volumes":[str(release/"pelias.json")+":/code/pelias.json:ro"],"depends_on":{"es":{"condition":"service_healthy"},"libpostal":{"condition":"service_started"}}})
        atomic_json(release/"compose.json",{"services":services,"networks":{"maps":{"external":True,"name":self.config["network"]}}})
        pelias={"esclient":{"hosts":[{"host":f"maps-{version}-es","port":9200}]},"schema":{"indexName":"pelias"},"api":{"indexName":"pelias","services":{"libpostal":{"url":f"http://maps-{version}-libpostal:4400"}}},"imports":{"adminLookup":{"enabled":False},"openstreetmap":{"datapath":"/data","leveldbpath":"/data/pelias-work","importVenues":True,"import":[{"filename":"region.osm.pbf"}]}},"logger":{"level":"info"}}
        atomic_json(release/"pelias.json",pelias)

    def download(self,url,target,algorithm,expected):
        if urllib.parse.urlparse(url).scheme!="https": raise ValueError("HTTPS download URL required")
        if target.exists() and self.digest(target,algorithm)==expected: return
        partial=target.with_suffix(target.suffix+".part")
        self.run(["curl","--fail","--location","--retry","3","--connect-timeout","15","--proto","=https","--proto-redir","=https",url,"--output",str(partial)])
        if self.digest(partial,algorithm)!=expected: raise ValueError("download checksum mismatch: "+str(target))
        os.replace(partial,target)

    @staticmethod
    def digest(path,algorithm="sha256"):
        digest=hashlib.new(algorithm)
        with Path(path).open("rb") as stream:
            for block in iter(lambda:stream.read(1<<20),b""): digest.update(block)
        return digest.hexdigest()

    def tool(self,release,image,arguments):
        self.run(["docker","run","--rm","--network",self.config["network"],"--memory",self.config["build_memory"],"--cpus",self.config["build_cpus"],"-v",str(release)+":/data","-v",str(HERE)+":/maps:ro","-e","PELIAS_CONFIG=/data/pelias.json",image,*arguments])

    def build(self,version,restart_from=None):
        release=self.release(version)
        release.mkdir(exist_ok=True)
        if (release/"validated.json").exists(): raise ValueError("validated releases are immutable; choose a new version")
        minimum=self.config.get("min_free_bytes")
        if not isinstance(minimum,int) or minimum<=0: raise ValueError("set min_free_bytes from a measured pilot build and required headroom")
        if shutil.disk_usage(self.root).free<minimum: raise ValueError("insufficient free disk space for configured build budget")
        endpoint=urllib.parse.urlparse(self.config["public_url"])
        if endpoint.scheme!="https" or not endpoint.hostname or endpoint.query or endpoint.fragment: raise ValueError("public_url must be a public HTTPS base URL")
        self.network()
        self.generate_compose(release)
        state=read_json(release/"state.json") if (release/"state.json").exists() else {"version":version,"started_at":now(),"completed":[]}
        fingerprint=hashlib.sha256(json.dumps({"config":self.config,"images":self.images,"schema":self.digest(HERE/"tilemaker.json"),"lua":self.digest(HERE/"process.lua")},sort_keys=True).encode()).hexdigest()
        if state.get("fingerprint",fingerprint)!=fingerprint: raise ValueError("build configuration changed; use a new version")
        state["fingerprint"]=fingerprint
        if restart_from and restart_from in state["completed"]:
            state["completed"]=state["completed"][:state["completed"].index(restart_from)]
        manifest={"version":version,"region":self.config["region"],"source":self.config["pbf_url"],"images":self.images,"bounds":self.config["bounds"]}
        with (release/"build.log").open("a") as self.log:
            try:
                for name,operation in [("download",lambda:self.source(release)),("tiles",lambda:self.tiles(release)),("osrm",lambda:self.graph(release)),("pelias",lambda:self.geocode(release)),("assets",lambda:self.assets(release)),("validation",lambda:self.validate(release))]:
                    if name in state["completed"]: continue
                    state.update({"stage":name,"status":"running","updated_at":now()}); atomic_json(release/"state.json",state)
                    started=time.monotonic()
                    operation()
                    state.setdefault("stage_duration_seconds",{})[name]=round(time.monotonic()-started,3)
                    state["completed"].append(name);atomic_json(release/"state.json",state)
                manifest.update({"validated_at":now(),"pbf_sha256":self.digest(release/"region.osm.pbf"),"artifact_bytes":sum(p.stat().st_size for p in release.rglob("*") if p.is_file())})
                atomic_json(release/"validated.json",manifest)
                state.pop("error",None)
                state.update({"status":"validated","updated_at":now()});atomic_json(release/"state.json",state)
            except Exception as error:
                state.update({"status":"failed","error":str(error),"updated_at":now()});atomic_json(release/"state.json",state)
                raise
            finally:
                self.log=None

    def source(self,release):
        if urllib.parse.urlparse(self.config["checksum_url"]).scheme!="https": raise ValueError("HTTPS checksum URL required")
        with urllib.request.urlopen(self.config["checksum_url"],timeout=30) as response:
            checksum=response.read(4096).decode().split()[0].lower()
        if not re.fullmatch(r"[a-f0-9]{32}",checksum): raise ValueError("invalid source MD5")
        self.download(self.config["pbf_url"],release/"region.osm.pbf","md5",checksum)

    def tiles(self,release):
        target=release/"map.mbtiles"
        if target.exists(): target.unlink() # unvalidated, failed stage only
        self.tool(release,self.images["tilemaker"],["/data/region.osm.pbf","--output","/data/map.mbtiles","--config","/maps/tilemaker.json","--process","/maps/process.lua","--store","/data/tilemaker-work"])

    def graph(self,release):
        for command in [["osrm-extract","-p","/opt/car.lua","/data/region.osm.pbf"],["osrm-partition","/data/region.osrm"],["osrm-customize","/data/region.osrm"]]:
            self.tool(release,self.images["osrm"],command)

    def geocode(self,release):
        (release/"pelias-work").mkdir(exist_ok=True)
        os.chown(release/"pelias-work",1001,1001)
        storage=release/"elasticsearch";storage.mkdir(exist_ok=True);os.chown(storage,1000,0)
        self.compose(release,"up","-d","es","libpostal")
        self.wait_http(self.address(release.name,"es",9200)+"/_cluster/health",lambda value:value.get("status") in ("yellow","green"))
        # Repeat imports write stable OSM document IDs into this isolated index only.
        try:
            self.http(self.address(release.name,"es",9200)+"/pelias")
        except urllib.error.HTTPError as error:
            if error.code!=404: raise
            self.tool(release,self.images["pelias_schema"],["node","scripts/create_index.js"])
        self.tool(release,self.images["pelias_osm"],["npm","start"])
        # Some importer subprocess failures exit 0; an empty index is never success.
        self.wait_http(self.address(release.name,"es",9200)+"/pelias/_count",lambda value:value.get("count",0)>0,attempts=10)

    def assets(self,release):
        write_assets(release,self.config["public_url"],release.name)
        archive=release/"fonts.zip"
        self.download(self.images["fonts_url"],archive,"sha256","d117316544b43a5dde7ee761b36e17701e9f85574e181d76a74814240fdbaf34")
        fonts=release/"public"/"fonts";fonts.mkdir(exist_ok=True)
        with zipfile.ZipFile(archive) as bundle:
            for entry in bundle.infolist():
                if entry.filename.startswith("Noto Sans Regular/") and not entry.is_dir():
                    target=(fonts/entry.filename).resolve()
                    if not target.is_relative_to(fonts.resolve()): raise ValueError("unsafe font archive path")
                    target.parent.mkdir(exist_ok=True);target.write_bytes(bundle.read(entry))
        shutil.copyfile(HERE/"OFL.txt",fonts/"OFL.txt")

    def validate(self,release):
        import sqlite3
        with sqlite3.connect(f"file:{release/'map.mbtiles'}?mode=ro",uri=True) as database:
            if database.execute("PRAGMA integrity_check").fetchone()[0]!="ok": raise ValueError("MBTiles integrity failed")
            if database.execute("SELECT count(*) FROM tiles").fetchone()[0]==0: raise ValueError("empty tile set")
            z,x,y=self.config["probe"]["tile"]
            row=database.execute("SELECT tile_data FROM tiles WHERE zoom_level=? AND tile_column=? AND tile_row=?",(z,x,2**z-1-y)).fetchone()
            if row is None: raise ValueError("control tile missing")
            counts=layer_counts(row[0])
            for layer in self.config["probe"].get("required_layers",["road","building"]):
                if counts.get(layer,0)==0: raise ValueError("missing control tile layer: "+layer)
        for asset in ("style.json","sprite.json","sprite.png","sprite@2x.png","fonts/Noto Sans Regular/0-255.pbf","fonts/Noto Sans Regular/1024-1279.pbf"):
            if not (release/"public"/asset).is_file(): raise ValueError("missing asset: "+asset)
        audit_style(read_json(release/"public"/"style.json"),self.config["public_url"].rstrip("/")+"/releases/"+release.name)
        self.compose(release,"up","-d")
        self.probe(release)

    def probe(self,release):
        version=release.name
        route=self.config["probe"]["route"]
        coordinates=";".join(",".join(map(str,point)) for point in route)
        result=self.wait_http(self.address(version,"osrm",5000)+"/route/v1/driving/"+coordinates+"?overview=full&geometries=geojson&radiuses=500;500",lambda value:value.get("code")=="Ok" and len(value.get("routes",[]))>0)
        if result["routes"][0]["distance"]<=0 or len(result["routes"][0]["geometry"]["coordinates"])<2: raise ValueError("road route validation failed")
        es=self.address(version,"es",9200)
        if self.http(es+"/pelias/_count")["count"]==0: raise ValueError("Pelias index has no documents")
        base=self.address(version,"pelias",4000)
        self.wait_http(base+"/v1/search?"+urllib.parse.urlencode({"text":self.config["probe"]["search"],"size":3}),lambda value:len(value.get("features",[]))>0)
        lon,lat=self.config["probe"]["reverse"]
        self.wait_http(base+"/v1/reverse?"+urllib.parse.urlencode({"point.lat":lat,"point.lon":lon,"size":1}),lambda value:len(value.get("features",[]))>0)
        state_path=release/"state.json"
        if state_path.exists():
            state=read_json(state_path)
            state.pop("error",None)
            state.update({"stage":"validation","status":"validated","updated_at":now()})
            atomic_json(state_path,state)

    def backend_override(self,release):
        manifest=read_json(release/"validated.json")
        version=release.name
        environment={"TAXI_ROUTING_OSRM_URL":f"http://maps-{version}-osrm:5000","TAXI_ROUTING_DATA_VERSION":version,"TAXI_ROUTING_MAX_SNAP_METERS":"500","TAXI_ROUTING_CACHE_TTL":"5m","TAXI_ROUTING_TIMEOUT":"3s","TAXI_GEOCODER_PELIAS_URL":f"http://maps-{version}-pelias:4000","TAXI_MAPS_PUBLIC_URL":self.config["public_url"],"TAXI_MAPS_DATA_VERSION":version,"TAXI_MAPS_UPDATED_AT":manifest["validated_at"]}
        return {"services":{self.config["backend_service"]:{"environment":environment,"networks":["maps-release"]}},"networks":{"maps-release":{"external":True,"name":self.config["network"]}}}

    def deploy_backend(self,override):
        args=["docker","compose","--project-directory",str(REPOSITORY),"--env-file",str(REPOSITORY/self.config["backend_env_file"]),"-p",self.config["backend_project"]]
        for file in self.config["backend_compose_files"]: args.extend(["-f",str(REPOSITORY/file)])
        args.extend(["-f",str(override),"up","-d","--no-build","--no-deps","--force-recreate","--wait","--wait-timeout","120",self.config["backend_service"]])
        self.run(args)

    def activate(self,version):
        release=self.release(version)
        manifest=read_json(release/"validated.json")
        if manifest["bounds"]!=self.config["bounds"]: raise ValueError("release bounds differ from activation configuration")
        self.validate(release)
        override=self.root/("backend-"+version+".json")
        atomic_json(override,self.backend_override(release))
        previous=read_json(self.root/"active.json") if (self.root/"active.json").exists() else None
        atomic_json(self.root/"activation.json",{"status":"switching","target":version,"previous":previous,"started_at":now()})
        try:
            self.deploy_backend(override)
            container_ids=subprocess.check_output(["docker","ps","-q","--filter","label=com.docker.compose.project="+self.config["backend_project"],"--filter","label=com.docker.compose.service="+self.config["backend_service"]]).decode().split()
            if len(container_ids)!=1: raise ValueError("expected exactly one backend container for this deployment")
            address=self.inspect(container_ids[0])["NetworkSettings"]["Networks"][self.config["network"]]["IPAddress"]
            published=self.http(f"http://{address}:8080/api/v1/public/map/config")
            if published.get("data",{}).get("data_version")!=version: raise ValueError("backend does not expose the selected map release; deploy the updated API image")
        except Exception:
            if previous:
                self.deploy_backend(self.root/("backend-"+previous["version"]+".json"))
            else:
                original=self.root/"backend-original.json"
                atomic_json(original,{"services":{}})
                self.deploy_backend(original)
            atomic_json(self.root/"activation.json",{"status":"failed","target":version,"previous":previous,"at":now()})
            raise
        atomic_json(self.root/"active.json",{"version":version,"previous":previous["version"] if previous else None,"activated_at":now()})
        atomic_json(self.root/"activation.json",{"status":"complete","target":version,"at":now()})

    def prune(self):
        active=read_json(self.root/"active.json")
        protected={active["version"],active.get("previous")}
        releases=sorted([path for path in (self.root/"releases").iterdir() if path.is_dir() and (path/"validated.json").exists()],key=lambda path:(path/"validated.json").stat().st_mtime,reverse=True)
        keep=max(2,int(self.config["retain_releases"]))
        protected.update(path.name for path in releases[:keep])
        for release in releases:
            if release.name in protected: continue
            checked=self.release(release.name)
            self.compose(checked,"down")
            shutil.rmtree(checked)

def main():
    parser=argparse.ArgumentParser()
    parser.add_argument("action",choices=["build","validate","activate","rollback","update","prune","status","prepare-transfer"])
    parser.add_argument("--config",required=True)
    parser.add_argument("--version")
    parser.add_argument("--from-stage",choices=["download","tiles","osrm","pelias","assets","validation"])
    arguments=parser.parse_args()
    pipeline=Pipeline(read_json(arguments.config))
    with pipeline.lock():
        version=arguments.version or datetime.now(timezone.utc).strftime("%Y%m")
        if arguments.action=="status":
            for file in [pipeline.root/"active.json",pipeline.root/"activation.json",*pipeline.root.glob("releases/*/state.json")]:
                if file.exists(): print(file, file.read_text())
        elif arguments.action in ("build","update"):
            release=pipeline.release(version)
            if not (release/"validated.json").exists(): pipeline.build(version,arguments.from_stage)
            if arguments.action=="update": pipeline.activate(version);pipeline.prune()
        elif arguments.action=="validate": pipeline.validate(pipeline.release(version))
        elif arguments.action=="activate": pipeline.activate(version)
        elif arguments.action=="rollback":
            active=read_json(pipeline.root/"active.json")
            target=arguments.version or active.get("previous")
            if not target: raise ValueError("no previous release")
            pipeline.activate(target)
        elif arguments.action=="prepare-transfer":
            release=pipeline.release(version);read_json(release/"validated.json");pipeline.compose(release,"stop")
            print("Release stopped. Copy the entire release directory (including Elasticsearch) preserving ownership; use the same root path and pinned images on target, then validate and activate.")
        else: pipeline.prune()

if __name__=="__main__":
    try: main()
    except Exception as error:
        print(json.dumps({"operation":"maps.update","status":"failed","error":str(error),"at":now()}),file=sys.stderr)
        sys.exit(1)
