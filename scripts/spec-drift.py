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
"""Compare the committed openapi.json with netcup's live spec.

    scripts/spec-drift.py             report drift as Markdown
    scripts/spec-drift.py --update    also overwrite openapi.json
    scripts/spec-drift.py --upstream FILE   compare against a local copy

Exit status: 0 in sync (or updated), 1 drift found, 2 the spec could not be
fetched or read.

info.version is ignored when deciding whether there is drift: netcup bumps it
with every SCP release whether or not the API changed.
"""

import argparse
import json
import pathlib
import sys
import urllib.request

UPSTREAM_URL = "https://www.servercontrolpanel.de/scp-core/api/v1/openapi"
LOCAL = pathlib.Path(__file__).resolve().parent.parent / "openapi.json"

# Applied to the upstream text before comparing or saving, so the committed
# spec never carries the storage provider's hostname from netcup's examples.
SANITIZE = {"s3.anexia-it.com": "s3.example.com"}

# Keys whose changes are documentation only and never alter generated types.
DOC_KEYS = {"description", "summary", "example", "examples", "externalDocs"}
METHODS = {"get", "put", "post", "delete", "patch", "head", "options"}
MAX_LISTED = 80


def fetch(source):
    if source:
        text = pathlib.Path(source).read_text(encoding="utf-8")
    else:
        req = urllib.request.Request(UPSTREAM_URL, headers={"User-Agent": "go-netcup-scp-spec-drift"})
        with urllib.request.urlopen(req, timeout=60) as resp:
            text = resp.read().decode("utf-8")
    for old, new in SANITIZE.items():
        text = text.replace(old, new)
    return text


def walk(old, new, path, out):
    """Collect (path, kind, detail) for every leaf difference."""
    if old == new:
        return
    if isinstance(old, dict) and isinstance(new, dict):
        for key in sorted(set(old) | set(new)):
            child = f"{path}/{key.replace('/', '~1')}"
            if key not in old:
                out.append((child, "added", new[key]))
            elif key not in new:
                out.append((child, "removed", old[key]))
            else:
                walk(old[key], new[key], child, out)
    else:
        out.append((path, "changed", (old, new)))


def short(value, limit=90):
    text = json.dumps(value, ensure_ascii=False, sort_keys=True)
    return text if len(text) <= limit else text[: limit - 1] + "…"


def operations(spec):
    return {
        f"{method.upper()} {path}"
        for path, item in spec.get("paths", {}).items()
        for method in item
        if method in METHODS
    }


def bullet_list(title, items):
    if not items:
        return []
    lines = [f"### {title}", ""]
    lines += [f"- `{item}`" for item in sorted(items)]
    return lines + [""]


def report(old, new, diffs):
    old_ops, new_ops = operations(old), operations(new)
    old_schemas = set(old.get("components", {}).get("schemas", {}))
    new_schemas = set(new.get("components", {}).get("schemas", {}))

    def is_doc(path):
        # A DOC_KEYS name directly under "properties" is a field that merely
        # happens to be called e.g. "description", not documentation.
        parts = path.split("/")
        return any(
            part in DOC_KEYS and parts[i - 1] != "properties"
            for i, part in enumerate(parts)
        )

    functional = [d for d in diffs if not is_doc(d[0])]
    changed_schemas = {
        d[0].split("/")[3]
        for d in functional
        if d[0].startswith("/components/schemas/") and d[0].count("/") > 3
    }

    lines = [
        f"Committed spec: `{old['info']['version']}` (OpenAPI {old.get('openapi')})  ",
        f"Upstream spec: `{new['info']['version']}` (OpenAPI {new.get('openapi')})",
        "",
        f"{len(functional)} functional difference(s), {len(diffs) - len(functional)} documentation-only.",
        "",
    ]
    lines += bullet_list("Operations added", new_ops - old_ops)
    lines += bullet_list("Operations removed", old_ops - new_ops)
    lines += bullet_list("Schemas added", new_schemas - old_schemas)
    lines += bullet_list("Schemas removed", old_schemas - new_schemas)
    lines += bullet_list("Schemas changed", changed_schemas)

    if functional:
        lines += ["### Functional differences", ""]
        for path, kind, detail in functional[:MAX_LISTED]:
            if kind == "changed":
                lines.append(f"- changed `{path}`: `{short(detail[0], 40)}` → `{short(detail[1], 40)}`")
            else:
                lines.append(f"- {kind} `{path}`: `{short(detail)}`")
        if len(functional) > MAX_LISTED:
            lines.append(f"- … and {len(functional) - MAX_LISTED} more")
        lines.append("")

    lines += [
        "### To refresh",
        "",
        "```bash",
        "task spec-update   # fetch + sanitize openapi.json",
        "task generate",
        "task check",
        "```",
    ]
    return "\n".join(lines)


def main():
    parser = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    parser.add_argument("--update", action="store_true", help="overwrite openapi.json with the upstream spec")
    parser.add_argument("--upstream", metavar="FILE", help="read the upstream spec from FILE instead of the network")
    args = parser.parse_args()

    try:
        text = fetch(args.upstream)
        new = json.loads(text)
        old = json.loads(LOCAL.read_text(encoding="utf-8"))
        # Guard against an error page or a truncated download passing as a spec.
        if not new.get("paths") or "version" not in new.get("info", {}):
            raise ValueError("upstream document has no paths or no info.version")
    except Exception as err:  # noqa: BLE001 - any failure here is "could not check"
        print(f"spec-drift: cannot load spec: {err}", file=sys.stderr)
        return 2

    diffs = []
    walk(old, new, "", diffs)
    diffs = [d for d in diffs if d[0] != "/info/version"]

    if not diffs:
        print(f"In sync with upstream ({new['info']['version']}).")
        return 0

    print(report(old, new, diffs))
    if args.update:
        LOCAL.write_text(text, encoding="utf-8")
        print(f"\nWrote {LOCAL.name}.", file=sys.stderr)
        return 0
    return 1


if __name__ == "__main__":
    sys.exit(main())
