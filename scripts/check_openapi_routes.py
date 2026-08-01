#!/usr/bin/env python3
"""AC-7.d: verify docs/openapi.yaml covers 100% of routes registered in
internal/delivery/http/route/route.go. Parses route.go with regex (no fiber
runtime needed) and diffs (method, path) pairs against the spec.
"""
import re
import sys
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parent.parent
ROUTE_GO = ROOT / "internal/delivery/http/route/route.go"
OPENAPI = ROOT / "docs/openapi.yaml"

METHODS = {"GET", "POST", "PUT", "PATCH", "DELETE"}


def fiber_to_openapi_path(path: str) -> str:
    path = re.sub(r":(\w+)", r"{\1}", path)
    if len(path) > 1 and path.endswith("/"):
        path = path[:-1]
    return path


def parse_routes(src: str):
    prefixes = {"app": "", "api": ""}
    routes = set()
    for line in src.splitlines():
        line = line.strip()

        m = re.match(r'(\w+)\s*:=\s*(\w+)\.Group\("([^"]*)"', line)
        if m:
            var, base, path = m.groups()
            prefixes[var] = prefixes.get(base, "") + path
            continue

        m = re.match(r'(\w+)\.(Get|Post|Put|Patch|Delete)\("([^"]*)"', line)
        if m:
            var, method, path = m.groups()
            if var not in prefixes:
                continue
            full = (prefixes[var] + path) or "/"
            routes.add((method.upper(), fiber_to_openapi_path(full)))
    return routes


def parse_openapi(spec: dict):
    routes = set()
    for path, ops in spec.get("paths", {}).items():
        for method in ops:
            if method.upper() in METHODS:
                routes.add((method.upper(), path))
    return routes


def main():
    routes = parse_routes(ROUTE_GO.read_text())
    spec = yaml.safe_load(OPENAPI.read_text())
    spec_routes = parse_openapi(spec)

    missing = sorted(routes - spec_routes)
    extra = sorted(spec_routes - routes)

    if missing:
        print(f"MISSING from openapi.yaml ({len(missing)}):")
        for method, path in missing:
            print(f"  {method} {path}")
    if extra:
        print(f"EXTRA in openapi.yaml, not registered in route.go ({len(extra)}):")
        for method, path in extra:
            print(f"  {method} {path}")

    print(f"route.go routes: {len(routes)}, openapi routes: {len(spec_routes)}")
    if missing or extra:
        sys.exit(1)
    print("OK: openapi.yaml covers 100% of registered routes")


if __name__ == "__main__":
    main()
