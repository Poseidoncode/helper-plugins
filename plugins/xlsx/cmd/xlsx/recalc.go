package main

import (
	"flag"

	"github.com/xuri/excelize/v2"
)

// cmdRecalc evaluates every formula with the Excelize calc engine,
// marks the workbook for full recalculation on open, saves it, and
// reports results as JSON.
//
// Cached <v> values are NOT rewritten: the public Excelize API removes
// formulas when setting cell values, so writing caches would destroy
// the model. Instead the authoritative computed values are returned in
// the report's "values" map, and FullCalcOnLoad guarantees Excel /
// LibreOffice display correct values on open.
func cmdRecalc(args []string) int {
	fs := flag.NewFlagSet("recalc", flag.ContinueOnError)
	output := fs.String("output", "", "write recalculated file here instead of in place")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() < 1 {
		fail("usage: xlsx recalc FILE [--output OUT]")
		return 2
	}
	path := fs.Arg(0)
	f, err := openWorkbook(path)
	if err != nil {
		emitJSON(buildReport(path, nil, 0,
			[]Finding{{Type: "file_error", Detail: err.Error()}},
			nil, false,
			map[string]any{"performed": false, "reason": "unreadable",
				"engine": engineVersion()},
			"excelize_calc"))
		return 2
	}
	defer f.Close()

	sheets := f.GetSheetList()
	var findings []Finding
	formulaCount := 0
	values := map[string]string{}
	valuesTruncated := false
	for _, sheet := range sheets {
		cells, err := formulaCells(f, sheet)
		if err != nil {
			findings = append(findings, Finding{
				Type: "file_error", Sheet: sheet, Detail: err.Error()})
			continue
		}
		for _, fc := range cells {
			formulaCount++
			value, calcErr := f.CalcCellValue(sheet, fc.Cell)
			marker, detail, unsupported, ok := classifyResult(value, calcErr)
			if ok {
				key := sheet + "!" + fc.Cell
				if len(values) < maxValuesEmbedded {
					values[key] = value
				} else {
					valuesTruncated = true
				}
				continue
			}
			fd := Finding{
				Type: "error_value", Error: marker,
				Sheet: sheet, Cell: fc.Cell,
				Formula: fc.Formula, Detail: detail,
			}
			if unsupported {
				fd.Type = "unsupported_function"
			}
			findings = append(findings, fd)
		}
	}

	// Guarantee host applications display computed values on open.
	full := true
	if err := f.SetCalcProps(&excelize.CalcPropsOptions{FullCalcOnLoad: &full}); err != nil {
		fail("set calc props: %v", err)
		return 2
	}
	dest := path
	if *output != "" {
		dest = *output
	}
	if err := f.SaveAs(dest); err != nil {
		emitJSON(buildReport(path, sheets, formulaCount,
			append(findings, Finding{Type: "file_error", Detail: err.Error()}),
			nil, false,
			map[string]any{"performed": false, "reason": "save_failed",
				"engine": engineVersion()},
			"excelize_calc"))
		return 2
	}

	report := buildReport(path, sheets, formulaCount, findings,
		values, valuesTruncated,
		map[string]any{"performed": true, "engine": engineVersion(),
			"full_calc_on_load": true},
		"excelize_calc")
	emitJSON(report)
	if len(findings) > 0 {
		return 1
	}
	return 0
}

// engineVersion reports the Excelize module version from the build info.
