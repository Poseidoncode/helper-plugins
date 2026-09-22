# X8 Worked Case — Superstore multiformat conversion

Progressive case: one source workbook, four output formats, reverse
validation back to xlsx. Canonical template X8
(`pitfalls-index.md` §X8).

## Setup

Source: `superstore.xlsx`, sheet `Orders` (row count `N`, recorded
before anything else — the canary). Target outputs:

1. `orders.csv` — raw values, UTF-8, years as text (`FY2025`)
2. `orders_summary.json` — machine-readable aggregates
3. `orders_report.html` → PDF — human-readable report
4. `orders_rt.xlsx` — reverse validation: CSV re-ingested through
   the XML-template path back into xlsx

## Trace

1. **Canary.** Record `N` = Raw row count and the control totals
   (`SUM(Sales)`, `SUM(Profit)`) from the source.
2. **CSV.** Values only; force year columns to text so `2025` does
   not become `2,025`. Re-count rows: must equal `N`.
3. **JSON.** Aggregates computed from the CSV (document the query);
   cross-check against the control totals from step 1.
4. **HTML→PDF.** Render the summary tables; print to PDF via the
   browser/CLI toolchain available on the host. No new numbers are
   introduced at this step — it is presentation only.
5. **Reverse validation.** Rebuild an xlsx from the CSV using the
   raw-XML path (`unpack.py` a minimal template, inject rows,
   `pack.py`), re-add the `=SUM`/`=SUMIF` summary formulas, run
   `recalc.py` → `status == "success"`, `total_errors == 0`,
   `total_formulas > 0`.
6. **Reconcile.** Control totals from the rebuilt workbook must
   equal step 1 exactly; row count must equal `N`.

## Pass criteria

- Row count `N` identical across xlsx → csv → xlsx.
- Control totals identical to the source at every hop.
- Final `orders_rt.xlsx` passes the global gates with live
   formulas, not pasted aggregates.
