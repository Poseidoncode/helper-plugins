# Recalc Guide — `scripts/recalc.py` reference

## 1. What it does

`recalc.py` is the mandatory final step before delivery. It
recalculates every formula and reports workbook health as a single
JSON object on stdout (diagnostics go to stderr):

```bash
python scripts/recalc.py mau_forecast.xlsx 30
```

```json
{
  "status": "success",
  "total_errors": 0,
  "total_formulas": 42
}
```

Pipeline: static raw-XML pre-scan → LibreOffice headless
`--convert-to` recalculation (when available) → raw-XML scan of the
*recalculated* bytes → in-place atomic replace (or `--output`) →
JSON report. Full pipeline detail lives in the script docstring.

## 2. JSON schema

| Key | Type | Meaning |
|---|---|---|
| `status` | string | `success` / `errors_found` / `error` (unreadable file, recalc failure) |
| `file` | string | Input path as given |
| `sheets_checked` | string[] | Sheets actually scanned |
| `total_formulas` | int | Distinct formula cells (shared-formula consumers counted once) |
| `total_errors` | int | All findings (hard errors + heuristic warnings) |
| `shared_formula_ranges` | int | Shared-formula definition ranges |
| `error_summary` | object | Marker/type → `{count, locations[]}` (≤20 locations each, then `truncated: true`) |
| `errors` | array | Full findings (capped at 100, see `errors_truncated`) |
| `errors_truncated` | bool | True when `errors` was capped |
| `recalc` | object | `{performed, reason, libreoffice}` — see §4 |
| `scanner` | string | Always `raw_xml` (see §10) |
| `compatibility_hint` | string | `raw_xml_only` after a LibreOffice rewrite, else `none` (see §10) |

Exit codes: `0` = success, `1` = errors_found, `2` = error.

## 3. The seven error markers

`#REF!` `#DIV/0!` `#VALUE!` `#NAME?` `#NULL!` `#NUM!` `#N/A`.
`error_summary` is keyed by marker for error-value cells
(`Model!C3`-style locations), plus structural keys:
`broken_sheet_ref`, `unknown_name_ref` (heuristic — verify manually),
`malformed_error_cell`, `file_error`.

## 4. The `recalc` object

| `performed` | `reason` | Meaning |
|---|---|---|
| true | `libreoffice_headless_convert` | Full dynamic recalculation done |
| false | `static_only_flag` | `--static-only` given; cached values scanned |
| false | `libreoffice_not_found` | Graceful degradation: static scan only |
| false | `libreoffice_failed` | `status` is `error`; see `errors[0].message` |

A static-only `success` is weaker than a recalculated one: runtime
errors (`#DIV/0!` on empty denominators) hide in stale caches. Always
rerun on a host with LibreOffice before delivery (§8).

## 5. Timeouts

Default 60s (second positional arg). Heavy workbooks hang LibreOffice:
raise to 180 (`recalc.py file.xlsx 180`); if it persists, the file
likely contains constructs LibreOffice chokes on (thousands of
volatile functions, external links) — split the model or pre-compute
the volatile section. Never wrap in `gtimeout`/`timeout` yourself;
the script already bounds the subprocess. (macOS `coreutils` is not
required.)

## 6. Sandboxed environments (AF_UNIX)

Every invocation routes through `office.soffice.get_soffice_env()`
(PATH augmentation for the macOS bundle, headless-friendly vars).
If the host sandbox denies `AF_UNIX` sockets, LibreOffice fails fast
with an actionable error instead of hanging — retry outside the
sandbox or on an unrestricted host. No compiler toolchain is needed;
no `LD_PRELOAD` shim is bundled.

## 7. Batch jobs

For hundreds of workbooks per hour, per-call `soffice` spawn
dominates. Options: keep the machine warm with an
[`unoserver`](https://github.com/unoconv/unoserver) listener and call
it instead of `recalc.py`'s engine step, or shard across hosts. The
JSON contract stays the same — validate downstream against
`total_errors` / `total_formulas`, not against wall-clock time.

## 8. CI without LibreOffice

`--static-only` gives you the structural checks (broken refs, stale
error markers, formula count gate) with zero dependencies — the
vendored engine is stdlib-only. Gate merges on
`status != "errors_found"`; schedule the full dynamic recalculation
on a LibreOffice host before release. `xlcalculator` /
[`formulas`](https://github.com/vinci1it2000/formulas) are a
pure-Python alternative for dynamic evaluation, but neither supports
array formulas, pivot tables, or custom functions.

## 9. LibreOffice rewrites and openpyxl

LibreOffice serializes `<mergeCell>`, drawings, and some number
formats in ways openpyxl's default parser cannot read
(`TypeError: expected <class 'int'>`). The file is intact and opens
in Excel. After any successful in-place recalculation this skill sets
`compatibility_hint: "raw_xml_only"`: avoid downstream openpyxl
rewrites; use the raw-XML workflow (`raw-xml-escape-hatch.md`). For
readback, `load_workbook(path, read_only=True, data_only=True)`
(streaming reader skips the merged-cell parser) also works.

## 10. Why `scanner` is always `raw_xml`

Both scans (pre and post) run through the vendored `formula_check`
engine: `zipfile` + `ElementTree` over the worksheet XML. No
openpyxl import, no cached-value trust, no format parsing — so vendor
XML, charts/drawings, and LibreOffice's own rewrites cannot break the
scan. There is no fallback because there is nothing to fall back
*from*; treat the reported `total_formulas` / `total_errors` as
authoritative and do not retry blindly.
