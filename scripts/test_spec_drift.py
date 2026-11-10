#!/usr/bin/env python3
# Copyright 2026 Laurynas Četyrkinas <laurynas@digilol.net>
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Tests for spec-drift.py. Run: python3 -m unittest discover -s scripts"""

import contextlib
import copy
import importlib.util
import io
import json
import pathlib
import sys
import tempfile
import unittest

_spec = importlib.util.spec_from_file_location(
    "spec_drift", pathlib.Path(__file__).with_name("spec-drift.py")
)
drift = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(drift)

BASE = {
    "openapi": "3.0.3",
    "info": {"version": "2026.0101"},
    "paths": {
        "/api/v1/servers": {
            "get": {"summary": "List servers", "responses": {"200": {"description": "ok"}}},
            "parameters": [],
        }
    },
    "components": {
        "schemas": {
            "Policy": {
                "type": "object",
                "description": "A policy",
                "properties": {
                    "description": {"type": "string", "nullable": True},
                    "name": {"type": "string", "example": "web"},
                },
            }
        }
    },
}


def diff(old, new):
    out = []
    drift.walk(old, new, "", out)
    return out


class WalkTest(unittest.TestCase):
    def test_identical(self):
        self.assertEqual(diff(BASE, copy.deepcopy(BASE)), [])

    def test_added_removed_changed(self):
        new = copy.deepcopy(BASE)
        props = new["components"]["schemas"]["Policy"]["properties"]
        props["name"]["type"] = "integer"
        props["extra"] = {"type": "boolean"}
        del props["description"]
        got = {(path, kind) for path, kind, _ in diff(BASE, new)}
        prefix = "/components/schemas/Policy/properties"
        self.assertEqual(
            got,
            {
                (f"{prefix}/name/type", "changed"),
                (f"{prefix}/extra", "added"),
                (f"{prefix}/description", "removed"),
            },
        )

    def test_path_keys_are_escaped(self):
        new = copy.deepcopy(BASE)
        new["paths"]["/api/v1/apps"] = {"get": {}}
        self.assertEqual(diff(BASE, new)[0][0], "/paths/~1api~1v1~1apps")


class ReportTest(unittest.TestCase):
    def report(self, new):
        return drift.report(BASE, new, diff(BASE, new))

    def test_field_named_description_is_functional(self):
        # Regression: a schema *property* called "description" was once
        # counted as documentation, hiding real type changes.
        new = copy.deepcopy(BASE)
        new["components"]["schemas"]["Policy"]["properties"]["description"]["type"] = ["string", "null"]
        text = self.report(new)
        self.assertIn("1 functional difference(s), 0 documentation-only.", text)
        self.assertIn("### Schemas changed", text)
        self.assertIn("- `Policy`", text)

    def test_documentation_changes_are_not_functional(self):
        new = copy.deepcopy(BASE)
        policy = new["components"]["schemas"]["Policy"]
        policy["description"] = "Reworded"
        policy["properties"]["name"]["example"] = "db"
        new["paths"]["/api/v1/servers"]["get"]["summary"] = "List all servers"
        text = self.report(new)
        self.assertIn("0 functional difference(s), 3 documentation-only.", text)
        self.assertNotIn("### Schemas changed", text)

    def test_operations_and_schemas_added_and_removed(self):
        new = copy.deepcopy(BASE)
        new["paths"]["/api/v1/apps"] = {"post": {}, "parameters": []}
        del new["paths"]["/api/v1/servers"]
        new["components"]["schemas"]["App"] = {"type": "object"}
        text = self.report(new)
        self.assertIn("### Operations added\n\n- `POST /api/v1/apps`", text)
        self.assertIn("### Operations removed\n\n- `GET /api/v1/servers`", text)
        self.assertIn("### Schemas added\n\n- `App`", text)
        # "parameters" is a path-level key, not an HTTP method.
        self.assertNotIn("PARAMETERS", text)

    def test_long_reports_are_truncated(self):
        new = copy.deepcopy(BASE)
        for i in range(drift.MAX_LISTED + 5):
            new["components"]["schemas"][f"S{i:03}"] = {"type": "object"}
        self.assertIn("- … and 5 more", self.report(new))


class MainTest(unittest.TestCase):
    def run_main(self, local, upstream_text, *flags):
        with tempfile.TemporaryDirectory() as tmp:
            tmp = pathlib.Path(tmp)
            local_path = tmp / "openapi.json"
            local_path.write_text(json.dumps(local), encoding="utf-8")
            upstream_path = tmp / "upstream.json"
            upstream_path.write_text(upstream_text, encoding="utf-8")
            old_local, old_argv = drift.LOCAL, sys.argv
            drift.LOCAL = local_path
            sys.argv = ["spec-drift.py", "--upstream", str(upstream_path), *flags]
            out, err = io.StringIO(), io.StringIO()
            try:
                with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
                    status = drift.main()
            finally:
                drift.LOCAL, sys.argv = old_local, old_argv
            return status, out.getvalue(), local_path.read_text(encoding="utf-8")

    def test_version_only_change_is_in_sync(self):
        new = copy.deepcopy(BASE)
        new["info"]["version"] = "2026.0202"
        status, out, _ = self.run_main(BASE, json.dumps(new))
        self.assertEqual(status, 0)
        self.assertIn("In sync", out)

    def test_drift_exits_1_and_leaves_file_alone(self):
        new = copy.deepcopy(BASE)
        new["components"]["schemas"]["App"] = {"type": "object"}
        status, out, saved = self.run_main(BASE, json.dumps(new))
        self.assertEqual(status, 1)
        self.assertIn("- `App`", out)
        self.assertEqual(json.loads(saved), BASE)

    def test_update_writes_sanitized_spec(self):
        new = copy.deepcopy(BASE)
        new["components"]["schemas"]["Url"] = {"example": "https://s3.anexia-it.com/bucket"}
        status, _, saved = self.run_main(BASE, json.dumps(new), "--update")
        self.assertEqual(status, 0)
        self.assertNotIn("anexia", saved)
        self.assertEqual(json.loads(saved)["components"]["schemas"]["Url"]["example"], "https://s3.example.com/bucket")

    def test_sanitized_hostname_alone_is_not_drift(self):
        local = copy.deepcopy(BASE)
        local["components"]["schemas"]["Url"] = {"example": "https://s3.example.com/bucket"}
        upstream = json.dumps(local).replace("s3.example.com", "s3.anexia-it.com")
        self.assertEqual(self.run_main(local, upstream)[0], 0)

    def test_unusable_upstream_exits_2(self):
        for text in ("<html>503</html>", "{}", json.dumps({"info": {"version": "x"}, "paths": {}})):
            with self.subTest(text=text):
                status, _, saved = self.run_main(BASE, text, "--update")
                self.assertEqual(status, 2)
                self.assertEqual(json.loads(saved), BASE)


if __name__ == "__main__":
    unittest.main()
