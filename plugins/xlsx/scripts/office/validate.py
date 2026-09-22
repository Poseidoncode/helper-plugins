#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
"""office.validate — static validation entry point (no LibreOffice needed).

Runs the vendored ``engine/formula_check.py`` checks (error-value cells,
broken sheet references, unknown named ranges, shared-formula integrity,
malformed cells) and prints the standardized JSON report. This is the
static half of ``scripts/recalc.py``; use ``recalc.py`` when you also
need dynamic (post-evaluation) values.

Usage:
    python scripts/office/validate.py input.xlsx
    python scripts/office/validate.py input.xlsx --sheet Sales

Output: the engine's ``--report`` JSON on stdout.
Exit codes: 0 = no errors, 1 = errors found or unreadable file.
"""

import importlib.util
import json
import os
import sys


def _load_formula_check():
    here = os.path.dirname(os.path.abspath(__file__))
    path = os.path.join(os.path.dirname(os.path.dirname(here)),
                        "engine", "formula_check.py")
    spec = importlib.util.spec_from_file_location("xlsx_engine_formula_check",
                                                  path)
    if spec is None or spec.loader is None:
        raise RuntimeError("cannot load engine module: " + path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def main(argv: list | None = None) -> int:
    args = [a for a in (argv if argv is not None else sys.argv[1:])
            if a != "--json"]
    sheet = None
    positional = []
    i = 0
    while i < len(args):
        if args[i] == "--sheet" and i + 1 < len(args):
            sheet = args[i + 1]
            i += 2
        else:
            positional.append(args[i])
            i += 1
    if not positional:
        print("Usage: validate.py <input.xlsx> [--sheet NAME]", file=sys.stderr)
        return 1
    formula_check = _load_formula_check()
    results = formula_check.check(positional[0], sheet_filter=sheet)
    print(json.dumps(formula_check.build_report(results),
                     indent=2, ensure_ascii=False))
    return 1 if results.get("error_count", 0) > 0 else 0


if __name__ == "__main__":
    sys.exit(main())
