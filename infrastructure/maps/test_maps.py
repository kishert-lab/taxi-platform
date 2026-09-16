"""Regression tests for release safety and HTTP asset contracts; no Docker required."""
import io
import json
from pathlib import Path
import sqlite3
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parent))

import assets
import manage
import server

class ReleaseTests(unittest.TestCase):
    def setUp(self):
        self.directory=tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root=Path(self.directory.name)
        self.pipeline=manage.Pipeline({"root":str(self.root),"retain_releases":2,"network":"test","backend_service":"backend","public_url":"https://maps.example.test","bounds":[19,41,-169,82]})

    def published(self,version):
        release=self.pipeline.release(version);release.mkdir()
        manage.atomic_json(release/"validated.json",{"version":version,"validated_at":"2026-09-09T00:00:00Z","bounds":[19,41,-169,82]})
        assets.write_assets(release,"https://maps.example.test",version)
        return release

    def test_path_escape_and_lock(self):
        for name in ("../escape","", "a/b"):
            with self.assertRaises(ValueError):self.pipeline.release(name)
        with self.pipeline.lock():
            with self.assertRaises(BlockingIOError):
                with self.pipeline.lock():pass

    def test_failed_activation_restores_previous(self):
        self.published("old");self.published("new")
        manage.atomic_json(self.root/"active.json",{"version":"old"})
        deploy=[]
        def fail_new(path):
            deploy.append(path.name)
            if path.name=="backend-new.json":raise RuntimeError("candidate API failed")
        with patch.object(self.pipeline,"validate"),patch.object(self.pipeline,"deploy_backend",side_effect=fail_new):
            with self.assertRaises(RuntimeError):self.pipeline.activate("new")
        self.assertEqual(manage.read_json(self.root/"active.json")["version"],"old")
        self.assertEqual(deploy,["backend-new.json","backend-old.json"])

    def test_prune_keeps_active_and_previous(self):
        for version in ("one","two","three","four"):self.published(version)
        manage.atomic_json(self.root/"active.json",{"version":"one","previous":"two"})
        with patch.object(self.pipeline,"compose"):
            self.pipeline.prune()
        self.assertTrue((self.root/"releases/one").exists())
        self.assertTrue((self.root/"releases/two").exists())

    def test_immutable_http_assets_tiles_and_conditional_request(self):
        release=self.published("test")
        with sqlite3.connect(release/"map.mbtiles") as database:
            database.execute("CREATE TABLE tiles(zoom_level INTEGER,tile_column INTEGER,tile_row INTEGER,tile_data BLOB)")
            database.execute("INSERT INTO tiles VALUES(1,1,0,?)",(b'\x1f\x8btest',))
        with patch.object(server,"ROOT",self.root):
            captured=[]
            def response(status,headers):captured[:]=[status,dict(headers)]
            data=server.application({"PATH_INFO":"/releases/test/style.json","REQUEST_METHOD":"GET"},response)
            self.assertEqual(captured[0],"200 OK")
            style=json.loads(b"".join(data));assets.audit_style(style,"https://maps.example.test/releases/test")
            etag=captured[1]["ETag"]
            server.application({"PATH_INFO":"/releases/test/style.json","REQUEST_METHOD":"GET","HTTP_IF_NONE_MATCH":etag},response)
            self.assertEqual(captured[0],"304 Not Modified")
            data=server.application({"PATH_INFO":"/releases/test/tiles/1/1/1.pbf","REQUEST_METHOD":"GET"},response)
            self.assertEqual(captured[1]["Content-Encoding"],"gzip")
            self.assertEqual(b"".join(data),b'\x1f\x8btest')
            server.application({"PATH_INFO":"/releases/test/../../validated.json","REQUEST_METHOD":"GET"},response)
            self.assertEqual(captured[0],"404 Not Found")
            (release/"validated.json").unlink()
            server.application({"PATH_INFO":"/releases/test/style.json","REQUEST_METHOD":"GET"},response)
            self.assertEqual(captured[0],"404 Not Found")

    def test_style_rejects_external_dependencies(self):
        release=self.published("external")
        style=manage.read_json(release/"public/style.json")
        style["glyphs"]="https://external.example/fonts/{range}.pbf"
        with self.assertRaises(ValueError):assets.audit_style(style,"https://maps.example.test/releases/external")

if __name__=="__main__":unittest.main()
