---
name: xlsx
description: >-
  Spreadsheet skill — read, edit, create, and convert .xlsx/.xlsm/.csv/.tsv files.
  Trigger when a spreadsheet file is the primary input or output: editing columns, formulas, formatting, charting, cleaning messy data, or creating new spreadsheets.
  Not for Word/HTML/PDF deliverables even if tabular data is involved.
allowed-tools: [Bash, Read, Write, Edit]
version: 1.0.0
descriptions:
  zh-Hans: "读取、编辑、创建和转换表格文件，支持 xlsx、xlsm、csv、tsv、公式、格式、图表和数据清洗。"
license: MIT
---

# xlsx

Read, edit, create, and recalculate `.xlsx` / `.xlsm` / `.csv` /
`.tsv` files. Default pairing: **pandas for tabular data + openpyxl
for live formulas, styles, and charts**. Recalculation through
[`scripts/recalc.py`](scripts/recalc.py) (LibreOffice headless) is
mandatory before delivery — openpyxl writes formulas as strings and
never evaluates them. All scanning is raw worksheet XML (stdlib
only), so files that crash openpyxl are still handled.

> **Formula-first.** A spreadsheet without live formulas is just a
> CSV with a fancier extension. **Every computed value** — totals,
> averages, growth rates, ratios, cross-sheet references, percent
> changes, anything derivable from other cells — **must be a live
> `=…` formula**, not a pre-computed number. Hard-coded numbers belong
> only in the **Assumptions** block. When in doubt, write the formula.
>
> **❌ Anti-pattern.** `df["total"] = df["a"] + df["b"];
> df.to_excel(...)` ships static numbers — the workbook dies the
> moment any input changes. Pandas/polars load and clean **raw
> inputs** only; derived values go in as `=` via openpyxl.

> **Spreadsheet output only.** Word / slides / HTML / standalone
> scripts / DB pipelines / Google Sheets API → switch skills.

## Operational rules — read before doing anything

> **1. Match [`docs/pitfalls-index.md`](docs/pitfalls-index.md)
> FIRST.** 8 canonical templates (X1–X8) with match signatures,
> slots, and executable traces. Multiple matches → fuse, strictest
> verification wins. No match → Decision Tree below.

> **2. Formula-first is non-negotiable.** Static numbers where
> formulas were possible = failed delivery. Hardcode only the
> Assumptions block.

> **3. `total_formulas == 0` is a red flag.** A "success" with zero
> formulas is a static dump. Rewrite with `=` formulas — unless the
> user explicitly asked for a snapshot.

> **4. Never `df.sample(N)` / `df.head(N)` the raw sheet.**
> Down-sampling destroys every aggregation on top. Write the full
> row count; switch the writer (xlsxwriter, `write_only`) if slow.

> **5. Summaries are Excel-native** — pivot tables or `=SUMIFS` /
> `=COUNTIFS` over Raw. Never Python `groupby` written back as
> values. Edit a Raw row → recalc → totals move.

> **6. Slicers can't be authored by openpyxl** (GUID-bound pivot
> caches, no API). Paths: (a) template inheritance — write into a
> pre-wired `template.xlsx` named range; (b) raw-XML transplant via
> `scripts/office/unpack.py` + `pack.py`. No template → say so,
> never silently downgrade to a dropdown.

> **7. Don't suppress stderr.** On failure redirect to a log and
> grep: `python scripts/recalc.py f.xlsx 60 2>/tmp/recalc.log`.

> **8. Enforce stated ranges in the formula.**
> "Score 0–100" → `=ROUND(MIN(MAX(raw, 0), 100), 1)`. Spot-check
> `MIN()`/`MAX()` after recalc.

> **9. 500k+ rows: pandas/polars first**, never openpyxl cell
> walking. openpyxl only at the output boundary for formulas and
> final assembly.

> **10. Trust the raw-XML scan.** `recalc.py` never parses through
> openpyxl, so merged-cell rewrites and vendor XML can't break it.
> After a LibreOffice rewrite (`compatibility_hint: "raw_xml_only"`),
> avoid downstream openpyxl rewrites — use the raw-XML hatch.
> Details: [`docs/recalc-guide.md`](docs/recalc-guide.md) §9–§10.

## 1. Scope and When to Use

- Inspect a workbook → `engine/xlsx_reader.py --json`, then
  pandas/polars for analysis (X3).
- Create → openpyxl with live formulas (X1,
  [`docs/create-edit-guide.md`](docs/create-edit-guide.md)).
- Edit in place → flip **inputs**, rerun `recalc.py` (X2). Files
  with VBA / pivots / slicers / external links → raw-XML hatch,
  never openpyxl round-trip
  ([`docs/raw-xml-escape-hatch.md`](docs/raw-xml-escape-hatch.md)).
- Convert (`csv` ↔ `xlsx` ↔ `ods` ↔ `tsv`) → pyexcel/pandas for
  raw values, then X1 for any derived cells (X4).
- Read-only "just print the values" → `xlsx_reader.py`
  ([`docs/advanced-reference.md`](docs/advanced-reference.md) §5).

## 2. Decision Tree

```
Read tabular data only ─────────────> xlsx_reader + pandas/polars -> X3
Create a new workbook ──────────────> openpyxl + `=` formulas     -> X1
Edit existing (plain) ──────────────> openpyxl, flip inputs       -> X2
Edit existing (VBA/pivot/slicer) ───> unpack → edit → pack        -> hatch
Any derived cells in output ────────> live `=` formulas, never values -> §5
Very large file (500k+ rows) ───────> polars read, xlsxwriter out  -> X6
Recalculate (mandatory) ────────────> scripts/recalc.py           -> §4
Convert formats ────────────────────> pyexcel                     -> X4
No LibreOffice (CI) ────────────────> --static-only / validate.py -> §4
```

**Never** `pandas.to_excel` / `polars.write_excel` for derived
values — static numbers, no formulas.

## 3. Cookbook (minimum viable; full recipes in `docs/`)

```python
import pandas as pd
frame = pd.read_excel("mau_forecast.xlsx", engine="openpyxl")  # raw read
```

```python
from openpyxl import Workbook, load_workbook
from openpyxl.styles import Font
book, ws = Workbook(), None
ws = book.active
ws["A1"], ws["B1"], ws["C1"] = "Quarter", "MAU (mm)", "ARR (¥mm)"
ws["B2"] = 148; ws["B2"].font = Font(color="0000FF")   # blue: assumption
ws["C2"] = "=B2*27*0.85"                               # black: derived
book.save("mau_forecast.xlsx")
book = load_workbook("mau_forecast.xlsx")              # data_only=False!
book["Sheet"]["B2"] = 0.90                             # flip input, not result
book.save("mau_forecast.xlsx")
```

`data_only=True` is read-only safe only — saving deletes all
formulas. Large writes → xlsxwriter (`write_formula`); huge reads →
polars (write side stays openpyxl). Details:
[`docs/create-edit-guide.md`](docs/create-edit-guide.md),
[`docs/advanced-reference.md`](docs/advanced-reference.md).

## 4. Recalculation Routes

```bash
python scripts/recalc.py mau_forecast.xlsx 30
```

```json
{ "status": "success", "total_errors": 0, "total_formulas": 42 }
```

`errors_found` adds `error_summary` (marker → ≤20 locations).
Exit codes: `0` success · `1` errors · `2` unreadable/un recalculable.
No LibreOffice → graceful static scan (`recalc.performed: false`);
CI → `--static-only` or `scripts/office/validate.py`. Full JSON
schema, the seven markers, timeouts, sandbox notes:
[`docs/recalc-guide.md`](docs/recalc-guide.md).

## 5. Conventions (summary)

Black `000000` = formulas · Blue `0000FF` = hardcoded inputs ·
Green = same-workbook links · Red = cross-file links · Yellow fill =
assumptions under review. Years as text (`FY2025`), units in headers
(`ARR (¥mm)`), `0.0%`, `(123)` negatives. Every blue cell gets
`Source: <origin> | <as-of> | <owner>`. Full rationale, grammar,
examples, template checklist:
[`docs/conventions-guide.md`](docs/conventions-guide.md).

## 6. Reference Index

| File | Purpose |
|---|---|
| [`docs/pitfalls-index.md`](docs/pitfalls-index.md) | **Read first.** X1–X8 canonical templates |
| [`docs/superstore-multiformat-conversion-case.md`](docs/superstore-multiformat-conversion-case.md) | X8 worked multiformat case |
| [`docs/create-edit-guide.md`](docs/create-edit-guide.md) | openpyxl recipes, merged cells, charts, raw-XML readback |
| [`docs/recalc-guide.md`](docs/recalc-guide.md) | `recalc.py` schema, markers, timeouts, sandbox, CI |
| [`docs/conventions-guide.md`](docs/conventions-guide.md) | Colors, formats, formula construction, sources |
| [`docs/raw-xml-escape-hatch.md`](docs/raw-xml-escape-hatch.md) | No-go table + unpack/edit/pack loop |
| [`docs/advanced-reference.md`](docs/advanced-reference.md) | polars/duckdb/xlcalculator/pyexcel/large-file/CI |
| [`docs/windows-tool-bootstrap.md`](docs/windows-tool-bootstrap.md) | Git Bash detection + installs |
| `scripts/recalc.py` | Recalculation entry point — mandatory final step |
| `scripts/office/soffice.py` | Hardened LibreOffice runner (`--check`, env) |
| `scripts/office/validate.py` | Static validation (no LibreOffice needed) |
| `scripts/office/unpack.py` / `pack.py` | XML escape hatch (wrap frozen `engine/`) |
| `engine/` | Vendored MiniMax scripts, frozen (see `NOTICE.md`) |

## 7. Troubleshooting

| Symptom | Fix |
|---|---|
| `#REF!` after row insert | Fix shifted coordinates, rerun `recalc.py` |
| `#DIV/0!` | `IFERROR(num/denom, 0)` or `IF(denom=0, "", …)` |
| `#N/A` | Check key whitespace/case/type in lookup column |
| `soffice timed out` | Raise timeout (`recalc.py f.xlsx 180`); see recalc-guide §5 |
| AF_UNIX / `Address already in use` | Sandbox denial — retry off-sandbox; recalc-guide §6 |
| `iter_rows()` → `None` | Merged region: unmerge, broadcast anchor, re-merge |
| Saved file lost all formulas | Was opened `data_only=True` — reload default, never save that handle |
| openpyxl `TypeError` on `<mergeCell>` | LibreOffice rewrite — file is fine; use hatch or `read_only` readback |
| polars output has no formulas | Expected — write side is always openpyxl |
| VBA/pivots/slicers stripped | Never openpyxl-round-trip these — hatch §1 |

## 8. Environment

| Dependency | Purpose | Install |
|---|---|---|
| Python 3.9+ (stdlib only for engine) | `recalc.py`, `office/*`, `engine/*` | system |
| `openpyxl` + `pandas` | Cookbook create/edit paths (§3) | `pip install openpyxl pandas` |
| `polars` / `xlsxwriter` / `pyexcel` (optional) | Large files / throughput / conversion | `pip install …` |
| LibreOffice (`soffice`) | Dynamic recalculation | `brew install --cask libreoffice` / `apt install libreoffice` |

`get_soffice_env()` hardening is transparent via `recalc.py`. Batch
jobs (100s/hour) → unoserver (recalc-guide §7). Windows → Git Bash
+ `winget` (docs/windows-tool-bootstrap.md); no LibreOffice →
GUI Ctrl+Shift+F9, then `--static-only` verify.

## Requirements

- Python 3.9+, stdlib only for `recalc.py` / `office/*` / `engine/*`
  (`openpyxl` + `pandas` only for the §3 cookbook paths)
- LibreOffice (`soffice` on PATH) for dynamic recalculation —
  static scan works without it
- External processes run as the current user (not a sandbox)
