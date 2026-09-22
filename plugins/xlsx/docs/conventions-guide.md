# Conventions Guide — formula-first Excel outputs

## 1. Rationale: why black-first

A reviewer scans a workbook asking one question: *"which numbers can I
trust to move when I change an input?"* The color code answers it at a
glance. Black (formula) is the default because most cells in a healthy
model are derived; blue (hardcoded input) is the exception that must
stand out. Any inversion — blue formulas, black assumptions — tells
the reviewer the opposite of the truth, which is worse than no color
at all.

## 2. Color coding

| Color | Hex | Meaning | Anti-pattern |
|---|---|---|---|
| Black | `000000` | Every formula and computed result | `=A1+B1` in blue looks like a manual input |
| Blue | `0000FF` | Hard-coded inputs and scenario assumptions | `=Sheet2!A1` in blue hides the cross-sheet link |
| Green | `008000` | Same-workbook cross-sheet links | Green cross-file link hides the external dependency |
| Red | `FF0000` | Cross-file external links | Red same-sheet `=A1` implies a fake dependency |
| Yellow fill | `FFFF00` bg | Critical assumptions awaiting review | Yellow everywhere drowns out what needs review |

Three- and six-digit hex are equivalent (`#000` ≡ `#000000`); both
openpyxl and xlsxwriter accept either form.

## 3. Hardcode-source grammar and examples

Every blue cell carries its provenance in the next column or as a cell
comment, in this grammar:

```text
Source: <origin> | <as-of> | <owner>
```

- `<origin>`: report name, URL, "user-supplied", "assumption"
- `<as-of>`: date or version the value was taken at
- `<owner>`: who can confirm or update it

Examples:

```text
Source: Q3 board deck p.12 | 2026-09-01 | CFO office
Source: user-supplied | session 2026-09-22 | requester
Source: assumption, needs review | v0.3 draft | analyst
Source: https://example.com/pricing | 2026-08-15 | vendor
Source: prior-year actuals, frozen | FY2025 close | finance
```

A workbook with undocumented blue cells fails review: the reviewer
cannot distinguish "measured" from "guessed".

## 4. Formula construction

1. **Reference, don't repeat.** `=B2*C2*D2`, never `=148*27*0.85`.
   Literals inside formulas are invisible assumptions.
2. **Enforce stated ranges.** "Score 0–100" →
   `=ROUND(MIN(MAX(raw, 0), 100), 1)` (skill rule 8). Clamp at the
   formula, not in a comment.
3. **Guard division.** `=IFERROR(num/denom, 0)` or
   `=IF(denom=0, "", num/denom)` — pick the zero/blank semantics the
   downstream aggregation expects (`SUM` tolerates both; `AVERAGE`
   does not tolerate `0`).
4. **Cross-sheet links stay visible.** Same-workbook links are green;
   the formula text keeps the sheet qualifier (`=Sales!B2`), never a
   bare value copied across sheets.
5. **One unit per column, in the header.** `ARR (¥mm)`, `Tokens (bn)`,
   `MAU (mm)`, `Token unit price 1.2x` — never per-cell unit suffixes.

## 5. Number formats

| Type | Format | Rationale |
|---|---|---|
| Year | text (`"FY2025"`) | Numeric `2025` renders as `2,025` |
| Currency | `$#,##0` / `¥#,##0`, unit in header | Unit lives once per column |
| Zero | `"$#,##0;($#,##0);-"` | Em-dash = "zero by intent", blank = missing |
| Percentage | `0.0%` | One decimal scans; two is false precision |
| Multiple | `0.0x` | EV/EBITDA, P/S, P/E |
| Negative | `(123)` parentheses | Accounting convention; `-` blends into grids |

## 6. Template preservation checklist

When updating someone else's template, existing choices override this
skill's defaults. Before editing, record:

- [ ] Body font and header font (name, size, bold)
- [ ] Column widths and row heights (do not auto-fit blindly)
- [ ] Merged regions (only the anchor cell carries the value)
- [ ] Number formats already applied
- [ ] Conditional formatting rules (note the ranges)
- [ ] Frozen panes, filters, print settings
- [ ] Named ranges and their scopes

After editing, diff the checklist: any deviation not requested by the
user is a defect.
