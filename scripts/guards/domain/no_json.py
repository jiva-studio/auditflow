#!/usr/bin/env python3
"""Checks that libs/domain has no JSON imports or struct tags."""

import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
DOMAIN_DIR = ROOT / "libs" / "domain"
FORBIDDEN = ('"encoding/json"', 'json:"')


def scan_file(file: Path) -> list[str]:
  lines = file.read_text(encoding="utf-8").splitlines()
  return [
      f"{file}:{idx} contains forbidden '{token}'"
      for idx, line in enumerate(lines, 1)
      if not line.strip().startswith("//")
      for token in FORBIDDEN
      if token in line
  ]


def main() -> None:
  files = [
      f for f in DOMAIN_DIR.rglob("*.go") if not f.name.endswith("_test.go")
  ]
  violations = [v for f in files for v in scan_file(f)]

  if violations:
    print("❌ [guard:domain:no_json] Violations found:")
    for v in violations:
      print(f"  • {v}")
    sys.exit(1)

  print("✅ [guard:domain:no_json] Passed")


if __name__ == "__main__":
  main()
