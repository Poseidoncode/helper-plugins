package main

import (
	"encoding/csv"
	"flag"
	"os"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// cmdConvert ingests a .csv/.tsv file as raw values into a new workbook.
// Aggregations are NOT computed here — add live = formulas afterwards
// (skill rule: summaries must be Excel-native).
func cmdConvert(args []string) int {
	fs := flag.NewFlagSet("convert", flag.ContinueOnError)
	sheet := fs.String("sheet", "Raw", "destination sheet name")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() < 2 {
		fail("usage: xlsx convert INPUT.csv OUTPUT.xlsx [--sheet Raw]")
		return 2
	}
	in, out := fs.Arg(0), fs.Arg(1)
	rows, cols, err := convertCSV(in, out, *sheet)
	if err != nil {
		fail("%v", err)
		return 2
	}
	emitJSON(map[string]any{
		"status": "success", "input": in, "output": out,
		"sheet": *sheet, "rows": rows, "cols": cols,
	})
	return 0
}

func convertCSV(in, out, sheet string) (rows, cols int, err error) {
	fh, err := os.Open(in)
	if err != nil {
		return 0, 0, err
	}
	defer fh.Close()
	r := csv.NewReader(fh)
	r.FieldsPerRecord = -1
	if strings.HasSuffix(strings.ToLower(in), ".tsv") {
		r.Comma = '\t'
	}
	records, err := r.ReadAll()
	if err != nil {
		return 0, 0, err
	}
	f := excelize.NewFile()
	defer f.Close()
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return 0, 0, err
	}
	for i, rec := range records {
		if len(rec) > cols {
			cols = len(rec)
		}
		for j, val := range rec {
			cell, err := excelize.CoordinatesToCellName(j+1, i+1)
			if err != nil {
				return 0, 0, err
			}
			// Numeric-looking values stay numeric; years like FY2025
			// stay text automatically.
			if num, perr := strconv.ParseFloat(strings.TrimSpace(val), 64); perr == nil && val != "" {
				if err := f.SetCellValue(sheet, cell, num); err != nil {
					return 0, 0, err
				}
				continue
			}
			if err := f.SetCellStr(sheet, cell, val); err != nil {
				return 0, 0, err
			}
		}
	}
	if err := f.SaveAs(out); err != nil {
		return 0, 0, err
	}
	return len(records), cols, nil
}
