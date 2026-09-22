# Conventions Guide — formula-first Excel outputs

## 1. Rationale: why black-first

A reviewer scans asking: *"which numbers move when I change an
input?"* Black (formula) is the default because most cells in a
healthy model are derived; blue (hardcoded input) is the exception
that must stand out. Inverted colors state the opposite of the
truth — worse than no color at all.

## 2. Color coding

| Color | Hex | Meaning | Anti-pattern |
|---|---|---|---|
| Black | `000000` | Every formula and computed result | `=A1+B1` in blue looks manual |
| Blue | `0000FF` | Hard-coded inputs and scenario assumptions | `=Sheet2!A1` in blue hides the link |
| Green | `008000` | Same-workbook cross-sheet links | Green cross-file link hides the dependency |
| Red | `FF0000` | Cross-file external links | Red same-sheet `=A1` implies a fake dependency |
| Yellow fill | `FFFF00` bg | Critical assumptions awaiting review | Yellow everywhere drowns the signal |

Excelize style: `&excelize.Style{Font: &excelize.Font{Color: "0000FF"}}`
(`create-edit-guide.md` §3).

## 3. Hardcode-source grammar and examples

Every blue cell carries provenance next door or as a comment:

```text
Source: <origin> | <as-of> | <owner>
```

```text
Source: Q3 board deck p.12 | 2026-09-01 | CFO office
Source: user-supplied | session 2026-09-22 | requester
Source: assumption, needs review | v0.3 draft | analyst
Source: https://example.com/pricing | 2026-08-15 | vendor
Source: prior-year actuals, frozen | FY2025 close | finance
```

Undocumented blue cells fail review: measured vs guessed must be
distinguishable.

## 4. Formula construction

1. **Reference, don't repeat.** `=B2*C2*D2`, never `=148*27*0.85`.
2. **Enforce stated ranges.** "Score 0–100" →
   `=ROUND(MIN(MAX(raw, 0), 100), 1)` (skill rule 8).
3. **Guard division.** `=IFERROR(num/denom, 0)` or
   `=IF(denom=0, "", num/denom)` — match zero/blank semantics to
   what downstream `SUM`/`AVERAGE` expects.
4. **Keep cross-sheet links visible and green**, with the sheet
   qualifier in the formula text.
5. **One unit per column, in the header.** `ARR (¥mm)`,
   `Token unit price 1.2x` — never per-cell suffixes.
6. **Stay inside engine coverage.** Prefer SUM/IF/VLOOKUP-class
   functions over dynamic arrays; anything uncovered surfaces as
   `unsupported_function` at recalc (rule 10).

## 5. Number formats

| Type | Format | Rationale |
|---|---|---|
| Year | text (`"FY2025"`) | Numeric `2025` renders as `2,025` |
| Currency | `$#,##0` / `¥#,##0`, unit in header | Unit lives once per column |
| Zero | `"$#,##0;($#,##0);-"` | Em-dash = zero by intent, blank = missing |
| Percentage | `0.0%` | One decimal scans; two is false precision |
| Multiple | `0.0x` | EV/EBITDA, P/S, P/E |
| Negative | `(123)` parentheses | Accounting convention |

## 6. Template preservation checklist

Existing choices override skill defaults. Before editing, record:
body/header fonts; column widths and row heights; merged regions;
applied number formats; conditional formatting ranges; frozen
panes, filters, print settings; named ranges and scopes. Any
unrequested deviation after editing is a defect.
