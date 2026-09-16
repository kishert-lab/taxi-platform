"""Read-only WSGI origin; nginx owns TLS, CORS and caching. No business API traffic."""
import hashlib
import json
import mimetypes
import os
from pathlib import Path
import re
import sqlite3
from urllib.parse import unquote

ROOT = Path(os.environ.get("MAP_ROOT", "/data")).resolve()
PATH = re.compile(r"^/releases/([A-Za-z0-9_-]+)/(.+)$")
TILE = re.compile(r"^tiles/(\d+)/(\d+)/(\d+)\.pbf$")

def application(environ, start_response):
    path = unquote(environ.get("PATH_INFO", ""))
    if environ.get("REQUEST_METHOD") not in ("GET", "HEAD"):
        start_response("405 Method Not Allowed", [("Content-Length", "0")]); return []
    match = PATH.fullmatch(path)
    if not match:
        start_response("404 Not Found", [("Content-Length", "0")]); return []
    version, relative = match.groups()
    release = ROOT / "releases" / version
    headers = []
    try:
        # Publication marker is written only after validation.
        if not (release / "validated.json").is_file():
            raise FileNotFoundError("release is not validated")
        tile = TILE.fullmatch(relative)
        if tile:
            zoom, x, y = map(int, tile.groups())
            if not (0 <= zoom <= 14 and 0 <= x < 2**zoom and 0 <= y < 2**zoom):
                raise FileNotFoundError("invalid tile coordinates")
            with sqlite3.connect(f"file:{release / 'map.mbtiles'}?mode=ro&immutable=1", uri=True) as database:
                row = database.execute("SELECT tile_data FROM tiles WHERE zoom_level=? AND tile_column=? AND tile_row=?", (zoom, x, 2**zoom-1-y)).fetchone()
            if row is None:
                start_response("204 No Content", [("Cache-Control", "public,max-age=86400")]); return []
            content = row[0]
            mime = "application/vnd.mapbox-vector-tile"
            if content.startswith(b"\x1f\x8b"):
                headers.append(("Content-Encoding", "gzip"))
        else:
            asset = (release / "public" / relative).resolve()
            if not asset.is_relative_to((release / "public").resolve()) or not asset.is_file():
                raise FileNotFoundError("asset not found")
            content = asset.read_bytes()
            mime = {".pbf": "application/x-protobuf", ".json": "application/json", ".png": "image/png"}.get(asset.suffix, mimetypes.guess_type(str(asset))[0] or "application/octet-stream")
        etag = '"' + hashlib.sha256(content).hexdigest() + '"'
        headers.extend([("ETag", etag), ("Cache-Control", "public,max-age=31536000,immutable"), ("Content-Type", mime)])
        if environ.get("HTTP_IF_NONE_MATCH") == etag:
            start_response("304 Not Modified", headers); return []
        headers.append(("Content-Length", str(len(content))))
        start_response("200 OK", headers)
        return [] if environ.get("REQUEST_METHOD") == "HEAD" else [content]
    except FileNotFoundError:
        start_response("404 Not Found", [("Content-Length", "0"), ("Cache-Control", "no-store")]); return []
    except (OSError, sqlite3.Error) as error:
        print(json.dumps({"operation":"maps.serve", "error":str(error)}), flush=True)
        start_response("503 Service Unavailable", [("Content-Length", "0"), ("Cache-Control", "no-store")]); return []
