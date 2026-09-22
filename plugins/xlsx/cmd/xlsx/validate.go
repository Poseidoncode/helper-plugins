package main

import (
	"flag"
	"sort"

	"github.com/xuri/excelize/v2"
)

// cmdValidate runs static checks only: cached error markers, broken
// sheet references, unknown named-range references. No formula is
// evaluated — see recalc for dynamic verification.
func cmdValidate(args []string) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	sheetFilter := fs.String("sheet", "", "limit to one sheet")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() < 1 {
		fail("usage: xlsx validate FILE [--sheet NAME]")
		return 2
	}
	path := fs.Arg(0)
	f, err := openWorkbook(path)
	if err != nil {
		emitJSON(buildReport(path, nil, 0,
			[]Finding{{Type: "file_error", Detail: err.Error()}},
			nil, false,
			map[string]any{"performed": false, "reason": "unreadable"},
			"excelize_static"))
		return 2
	}
	defer f.Close()

	sheets := selectableSheets(f, *sheetFilter)
	validNames := map[string]bool{}
	for _, s := range f.GetSheetList() {
		validNames[s] = true
	}
	defined := map[string]bool{}
	for _, dn := range f.GetDefinedName() {
		defined[dn.Name] = true
	}

	var findings []Finding
	formulaCount := 0
	for _, sheet := range sheets {
		cells, err := formulaCells(f, sheet)
		if err != nil {
			findings = append(findings, Finding{
				Type: "file_error", Sheet: sheet, Detail: err.Error()})
			continue
		}
		for _, fc := range cells {
			formulaCount++
			// Cached error markers (written by Excel/LibreOffice).
			if v, err := f.GetCellValue(sheet, fc.Cell); err == nil && isErrorMarker(v) {
				findings = append(findings, Finding{
					Type: "error_value", Error: v,
					Sheet: sheet, Cell: fc.Cell, Formula: fc.Formula})
			}
			// Broken cross-sheet references.
			for _, ref := range extractSheetRefs(fc.Formula) {
				if !validNames[ref] {
					names := sortedKeys(validNames)
					findings = append(findings, Finding{
						Type: "broken_sheet_ref", Sheet: sheet,
						Cell: fc.Cell, Formula: fc.Formula,
						MissingSheet: ref,
						Detail:       "valid sheets: " + joinNames(names),
					})
				}
			}
			// Unknown named ranges (heuristic — verify manually).
			for _, name := range extractNameRefs(fc.Formula) {
				if !defined[name] {
					findings = append(findings, Finding{
						Type: "unknown_name_ref", Sheet: sheet,
						Cell: fc.Cell, Formula: fc.Formula,
						UnknownName: name, Heuristic: true,
					})
				}
			}
		}
	}

	report := buildReport(path, sheets, formulaCount, findings, nil, false,
		map[string]any{"performed": false, "reason": "static_only"},
		"excelize_static")
	emitJSON(report)
	if len(findings) > 0 {
		return 1
	}
	return 0
}

func selectableSheets(f *excelize.File, filter string) []string {
	var out []string
	for _, s := range f.GetSheetList() {
		if filter != "" && s != filter {
			continue
		}
		out = append(out, s)
	}
	return out
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func joinNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}
