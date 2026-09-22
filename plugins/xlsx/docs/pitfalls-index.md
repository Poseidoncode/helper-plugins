# Pitfalls Index — canonical query templates (X1–X8)

**Read this first.** Every "ship a wrong workbook" failure traces back to
skipping recalc, the `total_formulas` check, or the row-count canary.
Match the user's query against the Quick lookup table, copy the matching
template verbatim, substitute the `Slots`, and execute step by step.
Multiple partial matches → fuse: take the strictest verification from
each, never relax a constraint.

## Quick lookup

| ID | Task signature (user keywords) | Trace |
|---|---|---|
| X1 | create, new workbook, forecast, model, template from scratch | §X1 |
| X2 | edit, update, fix value, change assumption, existing file | §X2 |
| X3 | read, analyze, inspect, summarize, what is in this file | §X3 |
| X4 | convert, csv, tsv, ods, export, import | §X4 |
| X5 | error, `#REF!`, `#DIV/0!`, `#N/A`, broken, repair | §X5 |
| X6 | large, slow, 500k rows, huge file, memory | §X6 |
| X7 | pivot, slicer, filter dropdown, dashboard, chart | §X7 |
| X8 | superstore, multi-format, pdf report, reverse validation | §X8 + `superstore-multiformat-conversion-case.md` |

Global gates (apply to every template unless the user explicitly asked
for a static snapshot): `recalc.py` returns `status == "success"`,
`total_errors == 0`, and `total_formulas > 0` (rule 3).

---

## X1 — Create a new workbook

Match signatures: "create a forecast", "build a model for {COMPANY}",
"new spreadsheet with {METRICS}", "make an xlsx that computes …".

Slots: `{OUTPUT_XLSX}`, `{COMPANY}`, `{METRICS}`, `{ASSUMPTIONS}`.

Trace:

1. Put every user-supplied number in a leading **Assumptions** block,
   blue font (`0000FF`), with a `Source: …` annotation next door
   (`conventions-guide.md` §3).
2. Write every derived value as a live `=…` formula referencing the
   Assumptions block — never paste a computed number (rule 2).
3. Enforce stated ranges in the formula:
   `=ROUND(MIN(MAX(raw, 0), 100), 1)` for "score 0–100" (rule 8).
4. Run `python scripts/recalc.py {OUTPUT_XLSX} 60` → expect
   `status == "success"`, `total_errors == 0`, `total_formulas > 0`.
5. Spot-check `MIN()` / `MAX()` of each derived column after recalc.

Past failure: totals computed in pandas and written back as values —
the workbook looked right until the user edited an input.

## X2 — Edit an existing workbook in place

Match signatures: "update {FILE}", "change the take-rate to …",
"fix the labels in {SHEET}", "restyle".

Slots: `{FILE}`, `{SHEET}`, `{EDITS}`.

Trace:

1. Inspect first: `python engine/xlsx_reader.py {FILE} --json`
   (sheet names, headers, merged regions). Never modify the source
   before understanding it.
2. Flip **inputs**, not results. If the target cell holds a formula,
   trace its precedents and edit the assumption upstream.
3. If `load_workbook` (openpyxl) must be used: open with default
   `data_only=False`. `data_only=True` is read-only safe only —
   saving such a workbook deletes every formula.
4. If the file has VBA / pivot caches / slicers / external links:
   do NOT round-trip with openpyxl at all — use the raw-XML escape
   hatch (`raw-xml-escape-hatch.md`).
5. Rerun `recalc.py` → global gates. If `#REF!` appears, a formula
   still points at a shifted row — search, fix coordinate, rerun.

Past failure: saved with `data_only=True`, permanently replacing all
formulas with cached values.

## X3 — Read / analyze (no modification)

Match signatures: "analyze {FILE}", "what's in this workbook",
"summarize sales by region", "inspect".

Slots: `{FILE}`, `{QUESTION}`.

Trace:

1. `python engine/xlsx_reader.py {FILE} --json [--sheet NAME]` for
   structure discovery (never modify the source file).
2. Load tabular data with pandas/polars for analysis and QA.
3. If the user then wants the analysis *in* the workbook, switch to
   X2 and emit summaries as `=SUMIFS` / `=COUNTIFS` over the Raw
   sheet — never write `groupby` results back as values (rule 5).

Past failure: Python-side aggregation written back as static numbers,
silently breaking the edit→recalc→totals-move contract.

## X4 — Convert between formats

Match signatures: "convert {INPUT} to xlsx", "csv to excel",
"export as csv", "ods upload".

Slots: `{INPUT}`, `{OUTPUT}`.

Trace:

1. Format-unknown input → `pyexcel` (`save_book_as`); known tabular
   input → pandas read.
2. Conversion preserves **raw values only**. If the output needs
   totals/ratios, continue with X1: write raw rows, add `=…`
   formulas via openpyxl, recalc.
3. Year columns: force text (`"FY2025"`) so `2025` is not rendered
   as `2,025` (`conventions-guide.md` §2).
4. Rerun `recalc.py` on any `.xlsx` output → global gates.

Past failure: year `2025` thousands-separated to `2,025` in output.

## X5 — Repair formula errors

Match signatures: "#REF!", "#DIV/0!", "#N/A", "formulas broken",
"repair", "errors after recalc".

Slots: `{FILE}`.

Trace:

1. Run `python scripts/recalc.py {FILE} 60` and read
   `error_summary` (≤20 locations per marker).
2. `#REF!` → shifted/deleted rows: fix coordinates (X2 step 5).
   `#DIV/0!` → wrap denominator: `IFERROR(num/denom, 0)` or
   `IF(denom=0, "", num/denom)`. `#N/A` → check `VLOOKUP`/`MATCH`
   key whitespace, case, and type (`123` vs `"123"`).
3. Rerun `recalc.py` until `total_errors == 0`.
4. If openpyxl itself crashes on the file (merged-cell rewrites,
   vendor XML): trust the raw-XML scan — it never parses through
   openpyxl — and switch further edits to the raw-XML hatch.

Past failure: suppressing stderr (`2>/dev/null`) hid the only signal;
always redirect to a log file and grep instead (rule 7).

## X6 — Large files (500k+ rows / hundreds of MB)

Match signatures: "huge file", "500k rows", "too slow", "out of memory".

Slots: `{FILE}`, `{OUTPUT_XLSX}`.

Trace:

1. Read/filter/join with pandas/polars — never walk cells with
   openpyxl (too slow, encourages accidental sampling).
2. **Write the full row count** (rule 4). `df.sample(N)` /
   `df.head(N)` silently destroys every downstream aggregation.
3. Throughput-bound writes → xlsxwriter or
   `Workbook(write_only=True)` streaming
   (`advanced-reference.md` §6).
4. Derived values still go in as `=…` formulas; raise the recalc
   timeout (`recalc.py {FILE} 180`).
5. Row-count canary: compare Raw row count before/after; any
   `=SUMIF` summary must reconcile to the full count.

Past failure: 400k rows down-sampled to 100k "because openpyxl is
slow" — the summary was off by ~75% and looked plausible.

## X7 — Pivots, slicers, charts, dashboards

Match signatures: "pivot table", "slicer", "filter", "dashboard",
"add a chart".

Slots: `{FILE}`, `{FEATURE}`.

Trace:

1. Summaries the user can recompute → Excel-native: real pivot
   tables or `=SUMIFS` / `=COUNTIFS` over the Raw sheet (rule 5).
2. Slicers cannot be authored by openpyxl (no API for GUID-bound
   pivot-cache references). Two paths: (a) template inheritance —
   author `template.xlsx` once in Excel/LibreOffice, write into its
   named range; (b) raw-XML transplant via
   `scripts/office/unpack.py` + `pack.py`.
3. No template available → say so explicitly; never silently
   downgrade to a static dropdown (rule 6).
4. Charts via openpyxl are creatable but fragile across
   LibreOffice rewrites — prefer `compatibility_hint == "raw_xml_only"`
   handling after recalc.

Past failure: slicer silently replaced with a data-validation
dropdown; user discovered it only in the review meeting.

## X8 — Progressive multi-format case (Superstore)

Match signatures: "superstore", "multiple formats", "pdf report",
"round-trip validation".

See `superstore-multiformat-conversion-case.md` for the full worked
trace: Excel → CSV/JSON/HTML→PDF plus XML-template CSV→XLSX reverse
validation. Verification: every aggregation recomputed from the Raw
sheet after each format hop; row-count canary at every step.
