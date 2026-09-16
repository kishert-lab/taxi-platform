"""Generate a project-owned style and icon atlas. Glyphs are staged from pinned assets."""
import json
from pathlib import Path
import struct
import zlib

def write_assets(release: Path, public_url: str, version: str):
    root = release / "public"
    root.mkdir(exist_ok=True)
    base = public_url.rstrip("/") + "/releases/" + version
    source = {"type":"vector", "tiles":[base+"/tiles/{z}/{x}/{y}.pbf"], "minzoom":0, "maxzoom":14, "attribution":"© OpenStreetMap contributors"}
    layers = [{"id":"background","type":"background","paint":{"background-color":"#f5f2eb"}}]
    for name, color in [("landuse","#e3ecd9"),("water","#a9cee3"),("building","#d2c9bc")]:
        layers.append({"id":name,"type":"fill","source":"osm","source-layer":name,"paint":{"fill-color":color}})
    layers.append({"id":"roads","type":"line","source":"osm","source-layer":"road","paint":{"line-color":"#ffffff","line-width":["interpolate",["linear"],["zoom"],5,0.6,14,4,18,12]}})
    for name, field, minimum, size in [("road","name",12,12),("place","name",4,14),("housenumber","number",16,12),("poi","name",15,12)]:
        layout={"text-field":["get",field],"text-font":["Noto Sans Regular"],"text-size":size}
        if name=="road": layout["symbol-placement"]="line"
        if name=="poi": layout.update({"icon-image":"poi","text-offset":[0,1.2],"text-anchor":"top"})
        layers.append({"id":name+"-labels","type":"symbol","source":"osm","source-layer":name,"minzoom":minimum,"layout":layout,"paint":{"text-color":"#444a50","text-halo-color":"#ffffff","text-halo-width":1}})
    style={"version":8,"name":"Taxi OSM day","glyphs":base+"/fonts/{fontstack}/{range}.pbf","sprite":base+"/sprite","sources":{"osm":source},"layers":layers}
    (root/"style.json").write_text(json.dumps(style,ensure_ascii=False),encoding="utf-8")
    # Original simple circle icon, PNG generated without external image dependencies.
    def chunk(kind, payload):
        return struct.pack("!I",len(payload))+kind+payload+struct.pack("!I",zlib.crc32(kind+payload)&0xffffffff)
    for scale, suffix in [(1,""),(2,"@2x")]:
        size=16*scale
        pixels=b"".join(b"\0"+b"".join(bytes((55,110,155,255 if (x-size/2)**2+(y-size/2)**2<(5*scale)**2 else 0)) for x in range(size)) for y in range(size))
        png=b"\x89PNG\r\n\x1a\n"+chunk(b"IHDR",struct.pack("!2I5B",size,size,8,6,0,0,0))+chunk(b"IDAT",zlib.compress(pixels))+chunk(b"IEND",b"")
        (root/("sprite"+suffix+".png")).write_bytes(png)
        (root/("sprite"+suffix+".json")).write_text(json.dumps({"poi":{"x":0,"y":0,"width":size,"height":size,"pixelRatio":scale}}))
    (root/"LICENSE.txt").write_text("Map data © OpenStreetMap contributors, ODbL 1.0: https://www.openstreetmap.org/copyright\nStyle, schema and icons: project LICENSE. Fonts: Noto Sans, SIL Open Font License 1.1; see fonts/OFL.txt.\n",encoding="utf-8")
    audit_style(style,base)

def audit_style(style,base):
    if style.get("imports"): raise ValueError("style imports are not permitted")
    for key in ("glyphs","sprite"):
        if not style[key].startswith(base+"/"): raise ValueError("external style resource: "+key)
    for source in style["sources"].values():
        if "url" in source or any(not tile.startswith(base+"/") for tile in source.get("tiles",[])):
            raise ValueError("external tile source")
