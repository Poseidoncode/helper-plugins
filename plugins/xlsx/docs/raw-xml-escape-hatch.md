# Fidelity Notes — files Excelize round-trips carefully

Replaces the old raw-XML escape hatch: there is no unpack/pack loop
anymore, because Excelize preserves package parts (charts, drawings,
most metadata) across open→save. What remains is a verify list.

## 1. Care table: files needing human verification in Excel

| Artefact | Status | Path |
|---|---|---|
| VBA macros (`.xlsm`) | NOT supported — do not round-trip macro workbooks | Keep out of scope or edit in Excel |
| Pivot tables | Structure preserved; values NOT recalculated by the engine | Refresh in Excel, verify totals |
| Slicers | Cannot be authored; template-inherit only (rule 6) | Verify wiring in Excel |
| External connections / links | Preserved as XML; values depend on the source | Refresh in Excel |
| Array / dynamic-array formulas | NOT evaluated (`unsupported_function`) | Verify in Excel (rule 10) |
| Charts | Creatable (`AddChart`) and preserved | Verify rendering in Excel |

If any row except charts applies: prefer `read` (inspection is
always safe) and make edits in Excel, or restrict programmatic
edits to plain value cells and re-verify the artefact in Excel.

## 2. Safe operations on complex files

- Reading (`read`, `validate`) never modifies the source.
- Editing assumption cells and adding plain formulas is safe; the
  engine only touches the sheets/cells you address plus `calcPr`.
- `recalc` sets `FullCalcOnLoad`, so Excel recomputes everything —
  including pivots and array formulas — on open. The report's
  `values` map covers only engine-evaluated cells.

## 3. Slicer path (rule 6, template inheritance)

1. Author the slicer once in Excel against a named range;
   save as `template.xlsx`.
2. Open the template programmatically, write data into the named
   range, save as new.
3. Open the result in Excel and confirm the slicer is wired before
   delivery. No template → say so, never downgrade silently.
