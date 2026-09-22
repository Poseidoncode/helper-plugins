# Create / Edit Guide — Excelize recipes

All snippets assume `"github.com/xuri/excelize/v2"` imported as
`excelize`. Run as small Go programs (`go run`) or test harnesses —
the CLI covers read/validate/recalc/convert.

## 1. Minimal create (formula-first)

```go
f := excelize.NewFile()
defer f.Close()
f.SetSheetName("Sheet1", "Model")
f.SetCellValue("Model", "A1", "Quarter")
f.SetCellValue("Model", "B1", "MAU (mm)")
blue, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Color: "0000FF"}})
f.SetCellValue("Model", "B2", 148)
f.SetCellStyle("Model", "B2", "B2", blue) // blue: assumption
f.SetCellFloat("Model", "D2", 0.85, 4, 64)
f.SetCellFormula("Model", "E2", "=B2*C2*D2") // derived, references inputs
f.SaveAs("mau_forecast.xlsx")
// then: bin/xlsx recalc mau_forecast.xlsx
```

## 2. Minimal edit (flip inputs, not results)

```go
f, _ := excelize.OpenFile("mau_forecast.xlsx")
defer f.Close()
f.SetCellFloat("Model", "D2", 0.90, 4, 64) // assumption upstream of E2
f.Save()
// then: bin/xlsx recalc mau_forecast.xlsx
```

**Warning:** setting a value on a formula cell deletes the formula
(the API removes it). Write values only to assumption cells; if a
target cell holds a formula, trace precedents and edit upstream.

## 3. Styles and number formats

```go
style, _ := f.NewStyle(&excelize.Style{
    Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
    Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
    NumFmt: 165, // custom formats via NewCustomNumFmt / NewNumFmt
})
f.SetCellStyle("Model", "A1", "E1", style)
```

Color palette and format codes: `conventions-guide.md` §2/§5.

## 4. Merged cells

```go
f.MergeCell("Model", "A1", "E1")
f.UnMergeCell("Model", "A1", "E1")
```

Only the anchor (top-left) carries the value. Unmerge → broadcast
anchor → work → re-merge only if layout matters. Never place
formulas in non-anchor cells of a merged region.

## 5. Charts

```go
f.AddChart("Model", "G2", &excelize.Chart{
    Type: excelize.Line,
    Series: []excelize.ChartSeries{{
        Name: "Model!$C$1", Categories: "Model!$A$2:$A$13",
        Values: "Model!$C$2:$C$13",
    }},
    Title: []excelize.RichTextRun{{Text: "ARR"}},
})
```

Charts are creatable and round-trip — verify rendering in Excel
before delivery (X7).

## 6. Named ranges

Read: `f.GetDefinedName()`. Referencing an undefined name is
flagged by `validate` (`unknown_name_ref`, heuristic). Define names
in a template authored in Excel when cross-sheet models need them;
verify in Excel after programmatic edits.

## 7. Row insert / delete and `#REF!`

Use `InsertRows` / `RemoveRow`, then `recalc` and fix reported
locations. Structural edits shift coordinates — `#REF!` after an
insert means a formula still points at the old address.

## 8. Streaming writer (large files)

```go
sw, _ := f.NewStreamWriter("Raw")
sw.SetRow("A1", []any{"id", "amount"})
for _, rec := range records {
    sw.SetRow("A"+itoa(n), []any{rec.ID, rec.Amount})
}
sw.Flush()
f.SaveAs("big.xlsx")
```

Raw rows stream with near-constant memory; add the `=SUM`/`=SUMIF`
summary sheet afterwards and `recalc` (rule 4: full row count, X6).

## 9. Reading values back in Go

```go
v, _ := f.GetCellValue("Model", "E2")   // cached <v> (may be stale!)
fresh, _ := f.CalcCellValue("Model", "E2") // evaluated now
formula, _ := f.GetCellFormula("Model", "E2")
```

Prefer `CalcCellValue` (or the recalc `values` map) over cached
values — see recalc-guide §4.
