# Advanced Reference — alternatives and large-file / CI workflows

## 1. polars — read-side accelerator (500k+ rows)

```python
import polars as pl
frame = pl.read_excel("big.xlsx")
```

Polars is read-only leverage: filtering, joins, type normalization,
QA spot-checks. `write_excel` emits static numbers — no formulas, no
charts, no styles. Always pair with openpyxl on the write side and
emit derived values as `=` formulas (skill rule 9).

## 2. duckdb — SQL over files

```sql
SELECT region, SUM(amount) FROM read_csv('sales.csv') GROUP BY region;
```

Use duckdb to prototype the aggregation logic, then translate the
verified query into `=SUMIFS` / pivot-table definitions over the Raw
sheet — the workbook must recompute without duckdb installed.

## 3. xlcalculator / formulas — pure-Python evaluation

For CI runners without LibreOffice when cached values are untrusted
and `--static-only` is not enough. Limits: no array formulas, no
pivot tables, no custom functions. Delivery still requires a
`recalc.py` run on a LibreOffice host (`recalc-guide.md` §8).

## 4. pyexcel — format-agnostic conversion

```python
import pyexcel as pe
records = pe.get_records(file_name="upload.xlsx")   # csv / ods / tsv too
pe.save_book_as(file_name="raw.csv", dest_file_name="clean.xlsx")
```

Reach for it when the input format is unknown ahead of time.

## 5. extract-text — read-only inspection without code

```bash
python engine/xlsx_reader.py report.xlsx --json | head -50
python engine/xlsx_reader.py report.xlsx --sheet Sales --quality
```

Structure discovery, header lists, and data-quality audit without
modifying the source. This is the "user just wants the values
printed" path — no pandas, no workbook writes.

## 6. Large-file worked pattern

```text
polars read (filter/join/QA)
  → full row count preserved (rule 4: never sample)
  → xlsxwriter or write_only openpyxl for raw rows (throughput)
  → openpyxl =SUM/=SUMIF formulas on the summary sheet
  → recalc.py with raised timeout (180s)
  → row-count canary: Raw count vs summary reconciliation
```

Numbers: openpyxl cell-by-cell writes stall around six-figure rows;
xlsxwriter sustains them. LibreOffice recalc time grows with formula
count, not row count — keep volatile functions (`OFFSET`,
`INDIRECT`, whole-column refs like `A:A`) out of large models.

## 7. CI workflow

```bash
# structural gate, zero dependencies (engine is stdlib-only)
python scripts/office/validate.py model.xlsx
# or the static half of recalc:
python scripts/recalc.py model.xlsx --static-only
```

Gate merges on `status != "errors_found"`; run the full dynamic
recalculation on a LibreOffice host before release. See X6 for the
large-file trace and `recalc-guide.md` §8 for the static-vs-dynamic
contract.
