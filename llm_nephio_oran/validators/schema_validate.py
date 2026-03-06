from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any

from jsonschema import Draft202012Validator


def _load_json(path: Path) -> Any:
    return json.loads(path.read_text(encoding="utf-8"))


def validate_json_instance(schema_path: str, instance: dict[str, Any]) -> None:
    schema = _load_json(Path(schema_path))
    v = Draft202012Validator(schema)
    errors = sorted(v.iter_errors(instance), key=lambda e: e.path)
    if errors:
        msg = "\n".join([f"- {list(e.path)}: {e.message}" for e in errors])
        raise ValueError(f"Schema validation failed:\n{msg}")


def validate_json_file(schema_path: str, instance_path: Path) -> None:
    validate_json_instance(schema_path=schema_path, instance=_load_json(instance_path))


def main():
    p = argparse.ArgumentParser()
    p.add_argument("--schema", required=True)
    p.add_argument("--instance", required=True)
    args = p.parse_args()
    validate_json_file(args.schema, Path(args.instance))
    print("OK")


if __name__ == "__main__":
    main()
