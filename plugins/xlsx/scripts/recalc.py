#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
"""recalc.py — recalculate a workbook and report formula health as JSON.

Pipeline (all scanning is raw worksheet XML via the vendored engine, so
files that crash openpyxl — merged-cell rewrites, vendor XML, charts —
are still handled authoritatively):

1. Static pre-check of the input file (cached ``<v>`` values, broken
   sheet/name references, malformed cells).
2. If LibreOffice is available (and ``--static-only`` is not given):
   headless ``--convert-to`` recalculation into a temp file, then
   re-scan the *recalculated* file so runtime errors (``#DIV/0!``,
   ``#N/A`` …) that only surface after evaluation are caught.
3. Unless ``--output`` is given, atomically replace the input with the
   recalculated file (in-place delivery workflow).

Usage:
    python scripts/recalc.py mau_forecast.xlsx 30
    python scripts/recalc.py big_model.xlsx 180 --output /tmp/recalc.xlsx
    python scripts/recalc.py report.xlsx --static-only

Output: exactly one JSON object on stdout. Progress and diagnostics go
to stderr (never suppress it — see skill rule 7).

Exit codes:
    0 — status "success" (total_errors == 0)
    1 — status "errors_found"
    2 — status "error" (unreadable file, recalculation failure, …)

JSON schema — see docs/recalc-guide.md §2.
"""

import argparse
import importlib.util
import json
import os
import shutil
import sys
import tempfile

MAX_LOCATIONS_PER_MARKER = 20
MAX_ERRORS_EMBEDDED = 100

# Seven Excel error markers surfaced by recalculation.
ERROR_MARKERS = ("#REF!", "#DIV/0!", "#VALUE!", "#NAME?",
                 "#NULL!", "#NUM!", "#N/A")


def _skill_root() -> str:
    here = os.path.dirname(os.path.abspath(__file__))
    return os.path.dirname(here)  # scripts/ -> skill root


def _load_engine_module(name: str):
    path = os.path.join(_skill_root(), "engine", name + ".py")
    spec = importlib.util.spec_from_file_location("xlsx_engine_" + name, path)
    if spec is None or spec.loader is None:
        raise RuntimeError("cannot load engine module: " + path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def _load_soffice_helper():
    path = os.path.join(_skill_root(), "scripts", "office", "soffice.py")
    spec = importlib.util.spec_from_file_location("xlsx_office_soffice", path)
    if spec is None or spec.loader is None:
        raise RuntimeError("cannot load office.soffice: " + path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def log(msg: str) -> None:
    print(msg, file=sys.stderr)


def summarize_errors(errors: list) -> dict:
    """Group errors by marker/type, cap locations per key."""
    summary: dict = {}
    for e in errors:
        etype = e.get("type", "unknown")
        if etype == "error_value":
            key = e.get("error", "#UNKNOWN")
        else:
            key = etype
        loc = "%s!%s" % (e.get("sheet", "?"), e.get("cell", "?"))
        entry = summary.setdefault(key, {"count": 0, "locations": []})
        entry["count"] += 1
        if len(entry["locations"]) < MAX_LOCATIONS_PER_MARKER:
            entry["locations"].append(loc)
        else:
            entry["truncated"] = True
    return summary


def build_output(path: str, results: dict, recalc_info: dict,
                 scanner: str, compatibility_hint: str) -> dict:
    errors = results.get("errors", [])
    embedded = errors[:MAX_ERRORS_EMBEDDED]
    return {
        "status": ("success" if results.get("error_count", 0) == 0
                   else "errors_found"),
        "file": path,
        "sheets_checked": results.get("sheets_checked", []),
        "total_formulas": results.get("formula_count", 0),
        "total_errors": results.get("error_count", 0),
        "shared_formula_ranges": results.get("shared_formula_ranges", 0),
        "error_summary": summarize_errors(errors),
        "errors": embedded,
        "errors_truncated": len(errors) > len(embedded),
        "recalc": recalc_info,
        "scanner": scanner,
        "compatibility_hint": compatibility_hint,
    }


def fail_output(path: str, message: str, recalc_info: dict) -> dict:
    return {
        "status": "error",
        "file": path,
        "total_formulas": 0,
        "total_errors": 0,
        "error_summary": {},
        "errors": [{"type": "file_error", "message": message}],
        "recalc": recalc_info,
        "scanner": "raw_xml",
        "compatibility_hint": "none",
    }


def main(argv: list | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Recalculate a workbook and report formula health as JSON.")
    parser.add_argument("file", help="Workbook to recalculate (in-place "
                                     "unless --output is given).")
    parser.add_argument("timeout", nargs="?", type=int, default=60,
                        help="LibreOffice timeout in seconds (default: 60).")
    parser.add_argument("--output", default=None, metavar="OUT",
                        help="Write the recalculated file here instead of "
                             "replacing the input in place.")
    parser.add_argument("--static-only", action="store_true",
                        help="Skip LibreOffice; scan cached values only.")
    args = parser.parse_args(argv)

    target = os.path.abspath(args.file)
    if not os.path.isfile(target):
        out = fail_output(args.file, "input file not found: " + args.file,
                          {"performed": False, "reason": "missing_input",
                           "libreoffice": None})
        print(json.dumps(out, indent=2, ensure_ascii=False))
        return 2

    formula_check = _load_engine_module("formula_check")
    soffice = _load_soffice_helper()

    # Route every LibreOffice invocation through the hardened environment.
    os.environ.update({k: v for k, v in soffice.get_soffice_env().items()
                       if k in ("PATH", "SAL_USE_VCLPLUGIN")})

    recalc_info: dict = {"performed": False, "reason": "", "libreoffice": None}
    scan_path = target
    tmpdir = None

    if args.static_only:
        recalc_info["reason"] = "static_only_flag"
        log("recalc: static-only scan (LibreOffice step skipped)")
    else:
        binary = soffice.find_soffice()
        if not binary:
            recalc_info["reason"] = "libreoffice_not_found"
            log("recalc: LibreOffice not found — static scan of cached "
                "values only (install LibreOffice for dynamic recalculation)")
        else:
            version = soffice.libreoffice_version(binary)
            recalc_info["libreoffice"] = version
            log("recalc: LibreOffice %s — recalculating (timeout %ds)..."
                % (version, args.timeout))
            libre = _load_engine_module("libreoffice_recalc")
            tmpdir = tempfile.mkdtemp(prefix="xlsx_recalc_")
            recalc_out = os.path.join(tmpdir, "recalculated.xlsx")
            try:
                ok, message = libre.recalculate(target, recalc_out,
                                                timeout=args.timeout)
            except Exception as e:  # engine bug or hostile file, not a hang
                ok, message = False, "engine exception: %s" % e
            if not ok:
                detail = message if "not found" not in message.lower() \
                    else "LibreOffice disappeared between check and run"
                log("recalc: FAILED — " + detail.splitlines()[0])
                out = fail_output(args.file, detail, {
                    "performed": False, "reason": "libreoffice_failed",
                    "libreoffice": version})
                print(json.dumps(out, indent=2, ensure_ascii=False))
                shutil.rmtree(tmpdir, ignore_errors=True)
                return 2
            log("recalc: complete — " + message.splitlines()[0])
            recalc_info.update({"performed": True,
                                "reason": "libreoffice_headless_convert",
                                "libreoffice": version})
            scan_path = recalc_out

    # The scan is always raw worksheet XML (zipfile + ElementTree): it never
    # goes through openpyxl, so vendor XML / merged-cell rewrites that crash
    # openpyxl cannot break this step.
    try:
        results = formula_check.check(scan_path)
    except Exception as e:
        out = fail_output(args.file, "scan failed: %s" % e, recalc_info)
        print(json.dumps(out, indent=2, ensure_ascii=False))
        if tmpdir:
            shutil.rmtree(tmpdir, ignore_errors=True)
        return 2

    if results.get("errors") and any(
            e.get("type") == "file_error" for e in results["errors"]):
        out = fail_output(args.file,
                          results["errors"][0].get("message", "unreadable"),
                          recalc_info)
        print(json.dumps(out, indent=2, ensure_ascii=False))
        if tmpdir:
            shutil.rmtree(tmpdir, ignore_errors=True)
        return 2

    # Deliver the recalculated bytes.
    compatibility_hint = "none"
    if recalc_info["performed"]:
        dest = os.path.abspath(args.output) if args.output else target
        try:
            if os.path.abspath(scan_path) != dest:
                tmp_dest = dest + ".tmp-recalc"
                shutil.copy(scan_path, tmp_dest)
                os.replace(tmp_dest, dest)
        except OSError as e:
            out = fail_output(args.file, "cannot write output: %s" % e,
                              recalc_info)
            print(json.dumps(out, indent=2, ensure_ascii=False))
            shutil.rmtree(tmpdir, ignore_errors=True)
            return 2
        # LibreOffice rewrote the package XML: downstream openpyxl rewrites
        # may trip over its merged-cell / drawing serialization, so steer
        # the agent toward the raw-XML workflow (skill rule 10).
        compatibility_hint = "raw_xml_only"
        log("recalc: wrote recalculated file to " + dest)
    if tmpdir:
        shutil.rmtree(tmpdir, ignore_errors=True)

    out = build_output(args.file, results, recalc_info,
                       scanner="raw_xml",
                       compatibility_hint=compatibility_hint)
    print(json.dumps(out, indent=2, ensure_ascii=False))
    return 0 if out["status"] == "success" else 1


if __name__ == "__main__":
    sys.exit(main())
