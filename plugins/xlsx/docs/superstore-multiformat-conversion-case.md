# X8 Worked Case — Superstore multiformat conversion

Progressive case: one source workbook, four output formats, reverse
validation back to xlsx. Canonical template X8
(`pitfalls-index.md` §X8).

## Setup

Source: `superstore.xlsx`, sheet `Orders` (row count `N`, recorded
before anything else — the canary). Targets:

1. `orders.csv` — raw values, UTF-8, years as text
2. `orders_summary.json` — machine-readable aggregates
3. `orders_report.html` → PDF — human-readable report
4. `orders_rt.xlsx` — reverse validation: CSV re-ingested back
   into xlsx with live formulas

## Trace

1. **Canary.** `bin/xlsx read superstore.xlsx --sheet Orders`:
   record `N` and control totals (`SUM(Sales)`, `SUM(Profit)` —
   via `recalc` `values` or Excel).
2. **CSV.** Values only; year columns forced to text. Re-count
   rows: must equal `N`.
3. **JSON.** Aggregates computed from the CSV (document the
   method); cross-check against step 1 control totals.
4. **HTML→PDF.** Presentation only — no new numbers introduced.
5. **Reverse validation.** `bin/xlsx convert orders.csv
   orders_rt.xlsx`, re-add the `=SUM`/`=SUMIF` summary formulas
   (X1), `bin/xlsx recalc orders_rt.xlsx` → `success`,
   `total_errors == 0`, `total_formulas > 0`.
6. **Reconcile.** Control totals from the rebuilt workbook equal
   step 1 exactly; row count equals `N`.

## Pass criteria

- Row count `N` identical across xlsx → csv → xlsx.
- Control totals identical at every hop.
- Final `orders_rt.xlsx` passes the global gates with live
  formulas, not pasted aggregates.
