# Raw-XML Escape Hatch — surgical edits without openpyxl

## 1. No-go table: files openpyxl cannot safely round-trip

| Artefact | Symptom of round-tripping | Path |
|---|---|---|
| VBA macros (`.xlsm`) | Macro project stripped or corrupted | Hatch (or keep `.xlsm` out of scope) |
| Pivot caches | Cache definitions dropped; pivots go blank | Hatch |
| Slicers (`xl/slicers/*.xml`) | GUID-bound cache refs severed | Hatch (transplant only, never author) |
| External connections / links | Connection strings lost | Hatch |
| LibreOffice-rewritten mergeCells | `TypeError: expected <class 'int'>` on load | Hatch, or `read_only` readback |

If any row applies: do not `load_workbook` + `save`. Ever.

## 2. The loop

```bash
python scripts/office/unpack.py input.xlsx /tmp/work/
# edit XML under /tmp/work (see §3), or run engine helpers:
python engine/xlsx_insert_row.py /tmp/work --at 5 --formula "A5*2"
python scripts/office/validate.py /tmp/wrapped.xlsx   # after pack
python scripts/office/pack.py /tmp/work/ output.xlsx
python scripts/recalc.py output.xlsx 60
```

`unpack.py` pretty-prints the XML and reports high-risk content
(VBA, pivots, slicers) before you touch anything. `pack.py`
re-zips and validates well-formedness.

## 3. What to edit by hand (and what not to)

- Safe: `<v>` cached values (recalc overwrites anyway), `<f>`
  formula text, simple `<c>` insertions following existing patterns.
- Careful: shared formulas (`t="shared"`, `si` groups) — prefer
  `xlsx_insert_row.py` / `xlsx_shift_rows.py`, which keep them
  coherent (`create-edit-guide.md` §9).
- Do not hand-edit: `sharedStrings.xml` indices (use
  `engine/shared_strings_builder.py`), pivot cache XML, slicer XML
  beyond transplanting an intact part from a known-good template.

## 4. Slicer transplant (rule 6, path b)

1. Author the slicer once in Excel/LibreOffice against a named
   range; save as `template.xlsx`.
2. `unpack.py` both template and target.
3. Copy the intact `xl/slicers/*.xml` part + its `.rels` +
   `[Content_Types].xml` entries into the target tree.
4. `pack.py`, then open in Excel/LibreOffice and verify the slicer
   is wired before delivery.
