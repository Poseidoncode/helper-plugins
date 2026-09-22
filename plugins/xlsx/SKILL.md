---
name: xlsx
description: >-
  Spreadsheet skill — read, edit, create, and convert .xlsx/.xlsm/.csv/.tsv files
  with a pure-Go Excelize CLI (no Python, no LibreOffice).
  Trigger when a spreadsheet file is the primary input or output: editing columns, formulas, formatting, charting, cleaning messy data, or creating new spreadsheets.
  Not for Word/HTML/PDF deliverables even if tabular data is involved.
allowed-tools: [Bash, Read, Write, Edit]
version: 2.0.0
descriptions:
  zh-Hans: "读取、编辑、创建和转换表格文件，纯 Go Excelize 工具链，支持 xlsx、xlsm、csv、tsv、公式、格式、图表和数据清洗，无需 Python 与 LibreOffice。"
license: MIT
---

# xlsx

Read, edit, create, and recalculate `.xlsx` / `.xlsm` / `.csv` /
`.tsv` files with a **pure-Go CLI** (`cmd/xlsx`, built on
[Excelize](https://github.com/xuri/excelize)). No Python, no
LibreOffice, zero runtime dependencies beyond a Go toolchain:

```bash
cd plugins/xlsx
go build -o bin/xlsx ./cmd/xlsx     # once; bin/ is gitignored
bin/xlsx read mau_forecast.xlsx --sheet Model
bin/xlsx validate mau_forecast.xlsx
bin/xlsx recalc mau_forecast.xlsx
```

`recalc` evaluates every formula with Excelize's calc engine and
reports health as JSON — the mandatory gate before delivery.

> **Formula-first.** A spreadsheet without live formulas is just a
> CSV with a fancier extension. **Every computed value** — totals,
> averages, growth rates, ratios, cross-sheet references, percent
> changes, anything derivable from other cells — **must be a live
> `=…` formula**, not a pre-computed number. Hard-coded numbers belong
> only in the **Assumptions** block. When in doubt, write the formula.

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
> formulas is a static dump — unless the user explicitly asked for
> a snapshot. Fresh computed values live in the report's `values`
> map (`Sheet!Cell` → value).

> **4. Never sample the raw sheet.** Down-sampling destroys every
> aggregation on top. Write the full row count; for huge files use
> the streaming writer (`docs/advanced-reference.md` §6).

> **5. Summaries are Excel-native** — pivot tables or `=SUMIFS` /
> `=COUNTIFS` over Raw. Never values pasted back from another tool.
> Edit a Raw row → recalc → totals move.

> **6. Slicers can't be authored from scratch.** Neither Excelize
> nor any OOXML library wires GUID-bound pivot caches by hand.
> Path: author `template.xlsx` once in Excel, write into its named
> range, verify in Excel. No template → say so, never silently
> downgrade to a dropdown.

> **7. Don't suppress stderr.** JSON goes to stdout, diagnostics to
> stderr. On failure redirect to a log and grep:
> `bin/xlsx recalc f.xlsx 2>/tmp/recalc.log`.

> **8. Enforce stated ranges in the formula.**
> "Score 0–100" → `=ROUND(MIN(MAX(raw, 0), 100), 1)`. Spot-check
> computed `MIN()`/`MAX()` from the `values` map after recalc.

> **9. 500k+ rows: streaming first.** `NewStreamWriter` for writes,
> bounded `read --preview` for inspection. Never loop
> `GetCellFormula` over millions of cells interactively — use
> `recalc` (single in-memory pass) for full evaluation.

> **10. Respect the calc engine's limits.** Array formulas,
> iterative calculation, implicit/explicit intersection, table
> formulas, dynamic arrays (`FILTER`, `SORT`, `UNIQUE`…), and pivot
> calculation are NOT evaluated — they surface as
> `unsupported_function` findings, not error markers. Never ship
> such formulas without opening the file in Excel to verify.
> Details: [`docs/recalc-guide.md`](docs/recalc-guide.md) §7.

## 1. Scope and When to Use

- Inspect → `bin/xlsx read FILE [--sheet S] [--json]` (X3).
  CSV/TSV inputs read directly.
- Create → Go cookbook below (X1,
  [`docs/create-edit-guide.md`](docs/create-edit-guide.md)).
- Edit in place → flip **inputs**, rerun `recalc` (X2). Files with
  VBA / pivots / slicers / external links → check the fidelity
  notes first ([`docs/raw-xml-escape-hatch.md`](docs/raw-xml-escape-hatch.md)).
- Convert (`csv`/`tsv` → `xlsx`, raw values) → `bin/xlsx convert`
  (X4). Derived cells afterwards go in as `=` formulas (X1).
- Read-only "just print the values" → `read`
  ([`docs/advanced-reference.md`](docs/advanced-reference.md) §5).

## 2. Decision Tree

```
Read tabular data only ─────────────> xlsx read [--json]          -> X3
Create a new workbook ──────────────> Go cookbook + `=` formulas  -> X1
Edit existing (plain) ──────────────> flip inputs, recalc         -> X2
Edit existing (VBA/pivot/slicer) ───> fidelity notes, verify Excel-> hatch
Any derived cells in output ────────> live `=` formulas           -> §5
Very large file (500k+ rows) ───────> streaming writer            -> X6
Recalculate (mandatory) ────────────> xlsx recalc                 -> §4
Convert csv/tsv ────────────────────> xlsx convert                -> X4
```

## 3. Cookbook (Go; full recipes in `docs/`)

Create with live formulas (write once, run with `go run` or as a
test harness — the CLI covers read/validate/recalc/convert):

```go
package main

import "github.com/xuri/excelize/v2"

func main() {
    f := excelize.NewFile()
    defer f.Close()
    f.SetSheetName("Sheet1", "Model")
    f.SetCellValue("Model", "A1", "Quarter")
    f.SetCellValue("Model", "B1", "MAU (mm)")
    blue, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Color: "0000FF"}})
    f.SetCellValue("Model", "B2", 148)
    f.SetCellStyle("Model", "B2", "B2", blue)   // blue: assumption
    f.SetCellFloat("Model", "D2", 0.85, 4, 64)
    f.SetCellFormula("Model", "E2", "=B2*C2*D2") // black: derived
    f.SaveAs("mau_forecast.xlsx")
}
```

Edit in place — flip the input, never the result:

```go
f, _ := excelize.OpenFile("mau_forecast.xlsx")
defer f.Close()
f.SetCellFloat("Model", "D2", 0.90, 4, 64) // assumption upstream of E2
f.Save()
// then: bin/xlsx recalc mau_forecast.xlsx
```

Merging, charts, number formats, and the streaming writer:
[`docs/create-edit-guide.md`](docs/create-edit-guide.md),
[`docs/advanced-reference.md`](docs/advanced-reference.md).

## 4. Recalculation Routes

```bash
bin/xlsx recalc mau_forecast.xlsx
```

```json
{
  "status": "success",
  "total_errors": 0,
  "total_formulas": 42,
  "values": { "Model!E2": "3396.6" }
}
```

- Exit codes: `0` success · `1` errors_found · `2` operational error.
- `errors_found` adds `error_summary` (marker → ≤20 locations) and
  the full `errors` list (capped at 100).
- `values` holds freshly computed results for every formula cell
  (capped at 10k, see `values_truncated`) — authoritative, because…
- **Design note:** the public Excelize API removes formulas when
  setting cell values, so `recalc` does NOT rewrite cached `<v>`
  values. Instead it sets `FullCalcOnLoad`, guaranteeing Excel /
  LibreOffice display computed values on open, and returns the
  computed values in `values`. Never trust stale `<v>` caches;
  trust `values`.
- Error mapping: engine errors surface as Go errors and are
  classified to markers (`#DIV/0!`, `#N/A`, …). One known skew: a
  reference to a missing sheet reports `#NAME?` (Excel reports
  `#REF!`) — the finding still carries `missing_sheet`, treat it
  as a broken reference. `not support … function` → type
  `unsupported_function` (rule 10), never silently ignored.
- Static-only (no evaluation): `bin/xlsx validate FILE` — cached
  error markers, broken sheet/name refs. CI gate without running
  anything: `validate` has zero dependencies beyond the binary.
- Full schema, marker table, coverage list:
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
| [`docs/create-edit-guide.md`](docs/create-edit-guide.md) | Go recipes: styles, merge, charts, streaming |
| [`docs/recalc-guide.md`](docs/recalc-guide.md) | CLI schema, markers, coverage, FullCalcOnLoad design |
| [`docs/conventions-guide.md`](docs/conventions-guide.md) | Colors, formats, formula construction, sources |
| [`docs/raw-xml-escape-hatch.md`](docs/raw-xml-escape-hatch.md) | Fidelity notes for VBA/pivot/slicer files |
| [`docs/advanced-reference.md`](docs/advanced-reference.md) | Streaming, CSV, large-file, CI |
| [`docs/windows-tool-bootstrap.md`](docs/windows-tool-bootstrap.md) | Go toolchain install on Windows |
| `cmd/xlsx/` | CLI source (`read`, `validate`, `recalc`, `convert`) |
| `bin/xlsx` | Local build output (gitignored; `go build -o bin/xlsx ./cmd/xlsx`) |

## 7. Troubleshooting

| Symptom | Fix |
|---|---|
| `#REF!` after row insert | Fix shifted coordinates, rerun `recalc` |
| `#DIV/0!` | `IFERROR(num/denom, 0)` or `IF(denom=0, "", …)` |
| `#N/A` | Check key whitespace/case/type in lookup column |
| `unsupported_function` | Calc engine limit (rule 10) — verify in Excel, don't ship blind |
| Missing-sheet ref reports `#NAME?` | Known engine skew vs Excel's `#REF!` — read `missing_sheet`, fix the ref |
| `read` shows dimension `A1` on huge files | Cosmetic only if rows/cols look right; bounds are computed from content |
| `go build` fails: `go >= 1.25` | Excelize requires Go 1.25+ — upgrade the toolchain |
| Charts/pivots look off after save | Verify in Excel; see fidelity notes |
| CSV shows `2,025` for year 2025 | Force text years on ingest; conventions §5 |

## 8. Environment

| Dependency | Purpose | Install |
|---|---|---|
| Go 1.25+ | Build `bin/xlsx` (Excelize requirement) | macOS: `brew install go`; Windows: `winget install GoLang.Go` |
| `github.com/xuri/excelize/v2` | Only library dependency (BSD-3-Clause) | `go mod download` (pinned in `go.sum`) |

No Python. No LibreOffice. No C toolchain. `validate` runs anywhere
the binary runs; `recalc` evaluates in-process. Batch jobs
(100s/hour): single binary, no per-file process spawn beyond the one
you start. Windows → [`docs/windows-tool-bootstrap.md`](docs/windows-tool-bootstrap.md).

## Requirements

- Go 1.25+ toolchain (build time only)
- The `bin/xlsx` binary at skill root (`go build -o bin/xlsx ./cmd/xlsx`)
- Excel opens the final file for any `unsupported_function` case (rule 10)
