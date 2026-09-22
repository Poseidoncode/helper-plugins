# Pitfalls Index — canonical query templates (X1–X8)

**Read this first.** Every "ship a wrong workbook" failure traces back to
skipping recalc, the `total_formulas` check, or the row-count canary.
Match the user's query against the Quick lookup table, copy the matching
template verbatim, substitute the `Slots`, and execute step by step.
Multiple partial matches → fuse: take the strictest verification from
each, never relax a constraint.

Build once per session: `go build -o bin/xlsx ./cmd/xlsx` (from the
skill root). Global gates (unless the user asked for a static
snapshot): `bin/xlsx recalc` returns `status == "success"`,
`total_errors == 0`, `total_formulas > 0` (rule 3).

## Quick lookup

| ID | Task signature (user keywords) | Trace |
|---|---|---|
| X1 | create, new workbook, forecast, model, template from scratch | §X1 |
| X2 | edit, update, fix value, change assumption, existing file | §X2 |
| X3 | read, analyze, inspect, summarize, what is in this file | §X3 |
| X4 | convert, csv, tsv, export, import | §X4 |
| X5 | error, `#REF!`, `#DIV/0!`, `#N/A`, broken, repair | §X5 |
| X6 | large, slow, 500k rows, huge file, memory | §X6 |
| X7 | pivot, slicer, filter dropdown, dashboard, chart | §X7 |
| X8 | superstore, multi-format, pdf report, reverse validation | §X8 + `superstore-multiformat-conversion-case.md` |

---

## X1 — Create a new workbook

Match signatures: "create a forecast", "build a model for {COMPANY}",
"new spreadsheet with {METRICS}", "make an xlsx that computes …".

Slots: `{OUTPUT_XLSX}`, `{COMPANY}`, `{METRICS}`, `{ASSUMPTIONS}`.

Trace:

1. Write a small Go program (cookbook §3): every user-supplied
   number in a leading **Assumptions** block, blue font, with a
   `Source: …` annotation next door (`conventions-guide.md` §3).
2. Every derived value as `SetCellFormula` referencing the
   Assumptions block — never `SetCellValue` with a computed number
   (rule 2).
3. Enforce stated ranges in the formula (rule 8).
4. `bin/xlsx recalc {OUTPUT_XLSX}` → `success`, `total_errors == 0`,
   `total_formulas > 0`. Read fresh results from the `values` map —
   never from stale `<v>` caches.
5. Spot-check computed `MIN()`/`MAX()` of each derived column.

Past failure: totals pasted as values — correct until the first
input edit.

## X2 — Edit an existing workbook in place

Match signatures: "update {FILE}", "change the take-rate to …",
"fix the labels in {SHEET}", "restyle".

Slots: `{FILE}`, `{SHEET}`, `{EDITS}`.

Trace:

1. Inspect first: `bin/xlsx read {FILE} --sheet {SHEET}`
   (structure, headers, dimension). Never modify before
   understanding.
2. Flip **inputs**, not results. Trace precedents, edit the
   assumption upstream. Setting a value on a formula cell destroys
   the formula (Excelize removes it) — so write values only to
   assumption cells.
3. Files with VBA / pivot caches / slicers / external links: read
   the fidelity notes (`raw-xml-escape-hatch.md`) and verify the
   result in Excel.
4. Rerun `recalc` → global gates. `#REF!` means a formula still
   points at a shifted row — fix the coordinate, rerun.

Past failure: value written onto a formula cell, silently deleting it.

## X3 — Read / analyze (no modification)

Match signatures: "analyze {FILE}", "what's in this workbook",
"summarize sales by region", "inspect".

Slots: `{FILE}`, `{QUESTION}`.

Trace:

1. `bin/xlsx read {FILE} [--sheet NAME] [--json]` — structure
   discovery first, never modify the source.
2. Pull tabular data into the analysis tool of choice; CSV/TSV
   inputs read directly.
3. If the analysis must live *in* the workbook, switch to X2 and
   emit summaries as `=SUMIFS` / `=COUNTIFS` over the Raw sheet —
   never paste aggregates as values (rule 5).

Past failure: aggregates pasted as static numbers, breaking the
edit→recalc→totals-move contract.

## X4 — Convert between formats

Match signatures: "convert {INPUT} to xlsx", "csv to excel",
"export as csv".

Slots: `{INPUT}`, `{OUTPUT}`.

Trace:

1. `bin/xlsx convert {INPUT} {OUTPUT} [--sheet Raw]` — raw values
   only, numeric-looking cells stay numeric, years stay text.
2. Conversion preserves **raw values only**. Totals/ratios →
   continue with X1: add `=` formulas, recalc.
3. Rerun `recalc` on the `.xlsx` output → global gates.

Past failure: year `2025` rendered as `2,025`.

## X5 — Repair formula errors

Match signatures: "#REF!", "#DIV/0!", "#N/A", "formulas broken",
"repair", "errors after recalc".

Slots: `{FILE}`.

Trace:

1. `bin/xlsx recalc {FILE}` and read `error_summary` (≤20
   locations per marker) plus the `detail` field on each error.
2. `#REF!` → shifted/deleted rows: fix coordinates (X2 step 4).
   `#DIV/0!` → `IFERROR(num/denom, 0)` or
   `IF(denom=0, "", num/denom)`. `#N/A` → key
   whitespace/case/type. Missing-sheet refs surface as `#NAME?`
   (engine skew) — read `missing_sheet`, fix the ref.
3. `unsupported_function` → calc engine limit, not a formula bug:
   verify in Excel, don't ship blind (rule 10).
4. Rerun `recalc` until `total_errors == 0`.

Past failure: stderr suppressed, hiding the only signal (rule 7).

## X6 — Large files (500k+ rows / hundreds of MB)

Match signatures: "huge file", "500k rows", "too slow", "out of memory".

Slots: `{FILE}`, `{OUTPUT_XLSX}`.

Trace:

1. Inspect with bounded reads (`read --preview`, `--sheet`).
2. **Write the full row count** (rule 4) via the streaming writer
   (`advanced-reference.md` §6).
3. Derived values still go in as `=` formulas; `recalc` is a single
   in-memory pass — no per-file subprocess, no timeout tuning.
4. Row-count canary: Raw count before/after; summaries must
   reconcile to the full count.

Past failure: sampled 400k→100k rows; summary off by ~75%, looked
plausible.

## X7 — Pivots, slicers, charts, dashboards

Match signatures: "pivot table", "slicer", "filter", "dashboard",
"add a chart".

Slots: `{FILE}`, `{FEATURE}`.

Trace:

1. Recomputable summaries → real pivot tables or `=SUMIFS` /
   `=COUNTIFS` over Raw (rule 5). The calc engine does NOT evaluate
   pivots — verify them in Excel.
2. Slicers cannot be authored from scratch. Template inheritance:
   author `template.xlsx` once in Excel, write into its named
   range, verify in Excel.
3. No template → say so explicitly, never silently downgrade
   (rule 6).
4. Charts via `AddChart` are creatable — verify rendering in Excel
   before delivery.

Past failure: slicer silently replaced with a dropdown; discovered
in review.

## X8 — Progressive multi-format case (Superstore)

Match signatures: "superstore", "multiple formats", "pdf report",
"round-trip validation".

See `superstore-multiformat-conversion-case.md`: Excel → CSV/JSON/
HTML→PDF plus CSV→XLSX reverse validation (`convert` + formulas +
`recalc`). Row-count canary at every hop.
