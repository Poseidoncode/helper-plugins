# Recalc Guide — `xlsx recalc` reference

## 1. What it does

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

Pipeline: enumerate every formula cell → evaluate with the Excelize
calc engine → set `FullCalcOnLoad` → save (in place, or `--output`)
→ JSON report. No Python, no LibreOffice, no subprocess per file.

## 2. JSON schema

| Key | Type | Meaning |
|---|---|---|
| `status` | string | `success` / `errors_found` / `error` (unreadable, save failed) |
| `file` | string | Input path as given |
| `sheets_checked` | string[] | Sheets scanned |
| `total_formulas` | int | Formula cells evaluated |
| `total_errors` | int | All findings (markers + unsupported) |
| `error_summary` | object | Marker/type → `{count, locations[]}` (≤20 each, then `truncated`) |
| `errors` | array | Full findings (capped at 100, see `errors_truncated`) |
| `values` | object | Fresh results: `Sheet!Cell` → value (capped at 10k, see `values_truncated`) |
| `recalc` | object | `{performed, engine, full_calc_on_load}` |
| `scanner` | string | Always `excelize_calc` |

Exit codes: `0` = success, `1` = errors_found, `2` = error.

Each error carries `type` (`error_value` or `unsupported_function`),
`error` (marker), `sheet`, `cell`, `formula`, and `detail` (engine
message). `unsupported_function` additionally means rule 10 applies.

## 3. Error markers and engine skew

Standard markers surface as-is: `#REF!` `#DIV/0!` `#VALUE!`
`#NAME?` `#NULL!` `#NUM!` `#N/A`. Two known skews vs Excel:

- A reference to a **missing sheet** reports `#NAME?` (Excel
  reports `#REF!`). The finding carries the formula text — treat
  any `#NAME?` with a `Sheet!` ref as a broken reference.
- A missing lookup reports `#N/A` with detail
  (`VLOOKUP no result found`) — same marker as Excel.

## 4. Why `values`, not cached `<v>`

The public Excelize API removes a cell's formula when a value is set
on it, so rewriting caches would destroy the model. `recalc`
therefore leaves `<v>` untouched and returns authoritative results
in `values`. Consequences:

- **Never read results from the file after recalc** — read them
  from `values`.
- Stale `<v>` caches in the file are harmless: `FullCalcOnLoad`
  forces Excel/LibreOffice to recompute on open.
- `validate` (static) still inspects cached markers — meaningful
  only for files last saved by Excel/LibreOffice.

## 5. No timeouts to tune

Evaluation is an in-memory pass, not a subprocess. There is no
timeout flag; a pathological workbook fails fast with an engine
error instead of hanging. For 500k+ row files the pass is still
single-shot — see X6.

## 6. Sandboxes and CI

Nothing to sandbox-escape: no sockets, no child processes, no
compiler needed at runtime. `validate` and `recalc` run anywhere
the `bin/xlsx` binary runs. Gate merges on `status`; the binary is
`go build` output — vendor it or rebuild in CI from `go.mod`.

## 7. Coverage: what the engine evaluates

Broad support (SUM/SUMIFS, VLOOKUP/XLOOKUP/INDEX/MATCH, IF/IFS,
financial, text, date functions — the full list is in Excelize's
`CalcCellValue` docs). Explicitly **not** evaluated:

- Array formulas and dynamic arrays (`FILTER`, `SORT`, `UNIQUE`…)
- Iterative calculation (circular refs with iteration enabled)
- Implicit / explicit intersection
- Table formulas (structured references)
- Pivot-table calculation

These surface as `unsupported_function` with the engine message in
`detail` (e.g. `not support FILTER function`). They are never
silently passed: `total_errors` counts them, `status` goes
`errors_found`. Verify every such case in Excel before delivery
(rule 10).

## 8. Static validation (`xlsx validate`)

Cached error markers, broken sheet refs, unknown named ranges —
no evaluation. Same report shape (`scanner: excelize_static`,
`recalc.performed: false`). Use it for sub-second pre-checks and
for files whose formulas you do not intend to evaluate.
