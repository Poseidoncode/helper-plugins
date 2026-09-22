# Create / Edit Guide — openpyxl recipes and gotchas

## 1. Minimal create (formula-first)

```python
from openpyxl import Workbook
from openpyxl.styles import Font

book = Workbook()
ws = book.active
ws.title = "Model"
ws["A1"], ws["B1"], ws["C1"] = "Quarter", "MAU (mm)", "ARR (¥mm)"
ws["B2"] = 148
ws["B2"].font = Font(color="0000FF")          # blue: assumption
ws["C2"] = "=B2*27*0.85"                      # black: derived (default)
book.save("mau_forecast.xlsx")
```

New workbooks open fine in openpyxl because *you* control the XML.
Round-tripping *someone else's* file is where §7–§11 apply.

## 2. Minimal edit (flip inputs, not results)

```python
from openpyxl import load_workbook
book = load_workbook("mau_forecast.xlsx")     # data_only=False (default)
book["Model"]["B2"] = 0.90                    # assumption upstream of C2
book.save("mau_forecast.xlsx")
# then: python scripts/recalc.py mau_forecast.xlsx 60
```

## 3. `data_only=True` is read-only safe only

Opening with `data_only=True` replaces every formula object with its
cached value *in memory*; saving that workbook writes the values back
and the formulas are gone permanently. Rule: load twice if you need
both (values for QA, formulas for editing), never save the
`data_only` handle.

## 4. Charts (basics)

```python
from openpyxl.chart import LineChart, Reference
chart = LineChart()
chart.add_data(Reference(ws, min_col=3, min_row=1, max_row=13), titles_from_data=True)
ws.add_chart(chart, "E2")
```

Charts survive openpyxl round-trips of files *you* created, but a
LibreOffice rewrite may re-serialize drawings — after recalc, check
`compatibility_hint`; if `raw_xml_only`, do not re-save with
openpyxl or the chart may detach.

## 5. Conditional formatting

Keep rules few and range-exact; overlapping rules from repeated
script runs are the usual cause of "the file got slower". Prefer data
bars / color scales over per-cell fills you manage by hand.

## 6. Named ranges

```python
from openpyxl.workbook.defined_name import DefinedName
book.defined_names["TakeRate"] = DefinedName("TakeRate", attr_text="Model!$B$2")
```

`formula_check` flags references to undefined names
(`unknown_name_ref`, heuristic) — define before delivery.

## 7. Merged cells

Only the anchor (top-left) cell carries the value; the rest read as
`None`. Pattern: `unmerge_cells()` → broadcast the anchor value →
do your work → re-merge only if layout matters. Never write formulas
into non-anchor cells of a merged region.

## 8. Shared strings

openpyxl manages `sharedStrings.xml` transparently, but files with
hundreds of thousands of distinct strings bloat memory. For
write-heavy jobs prefer xlsxwriter; for surgical string-table edits
use `engine/shared_strings_builder.py`.

## 9. Row insert / delete and `#REF!`

Inserting or deleting rows shifts coordinates; formulas pointing at
moved rows go `#REF!`. After structural edits always run `recalc.py`
and fix the reported locations. For unpacked-XML workflows the engine
provides `xlsx_insert_row.py` (`--at`, `--formula`, `--copy-style-from`),
`xlsx_shift_rows.py`, and `xlsx_add_column.py`, which keep shared
formulas coherent — prefer them over manual XML edits.

## 10. `read_only=True` readback

`load_workbook(path, read_only=True, data_only=True)` streams values
without building the full DOM: it skips the merged-cell parser, so it
reads LibreOffice-rewritten files that crash the default parser.

## 11. Raw-XML pattern (formula strings without openpyxl)

When openpyxl cannot parse the file at all, extract formula text
directly from the worksheet XML:

```bash
python scripts/office/unpack.py broken.xlsx /tmp/work/
python - <<'EOF'
import xml.etree.ElementTree as ET
NS = "{http://schemas.openxmlformats.org/spreadsheetml/2006/main}"
tree = ET.parse("/tmp/work/xl/worksheets/sheet1.xml")
for c in tree.getroot().iter(NS + "c"):
    f = c.find(NS + "f")
    if f is not None and f.text:
        print(c.get("r"), "=", f.text)
EOF
```

This is the same scanner `recalc.py` uses internally
(`recalc-guide.md` §10). For the full unpack → edit → pack loop, see
`raw-xml-escape-hatch.md`.
