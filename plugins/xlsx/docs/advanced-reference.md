# Advanced Reference — streaming, CSV, large-file, CI

## 1. Streaming writer (canonical large-file path)

`NewStreamWriter` holds near-constant memory regardless of row
count. Pattern: stream Raw rows → `Flush` → add the summary sheet
with `=SUM`/`=SUMIF` formulas → `SaveAs` → `recalc`
(`create-edit-guide.md` §8, X6). Full row count always (rule 4).

## 2. CSV / TSV ingestion

`bin/xlsx convert IN.csv OUT.xlsx [--sheet Raw]`: numeric-looking
cells stay numeric, everything else stays text (years like `FY2025`
never become `2,025`). `.tsv` detected by extension. Conversion is
raw-values only — derived cells go in afterwards as `=` formulas
(X4 → X1).

## 3. Reading programmatically

- CLI: `read --json [--sheet S] [--preview N]` (bounded preview,
  safe on huge files).
- Go: `GetRows` for full scans, `CalcCellValue` for fresh results,
  `GetCellFormula` to audit derivations. Never trust cached `<v>`
  after programmatic writes (recalc-guide §4).

## 4. What replaced the old Python stack

| Old (v1.x) | New (v2.x) |
|---|---|
| pandas read | `read` / `GetRows` / CSV sniffing |
| openpyxl create/edit | Go cookbook (`create-edit-guide.md`) |
| `xlsxwriter` throughput | `NewStreamWriter` |
| pyexcel conversion | `convert` |
| LibreOffice recalc | in-process `recalc` (CalcCellValue) |
| `formula_check` static scan | `validate` |
| unpack/edit/pack surgery | fidelity notes (no longer needed) |

## 5. Large-file worked pattern

```text
bounded read (preview/headers)
  → stream full row count (never sample)
  → summary sheet with =SUM/=SUMIF
  → recalc (single in-memory pass)
  → row-count canary: Raw count vs summary reconciliation
```

Keep volatile functions (`OFFSET`, `INDIRECT`, whole-column refs)
out of large models — evaluation cost follows formula complexity.

## 6. CI workflow

```bash
go build -o bin/xlsx ./cmd/xlsx
bin/xlsx validate model.xlsx     # sub-second structural gate
bin/xlsx recalc model.xlsx       # full evaluation gate
```

Gate merges on `status`. One static binary, zero runtime
dependencies, no services to warm. See `recalc-guide.md` §6.
