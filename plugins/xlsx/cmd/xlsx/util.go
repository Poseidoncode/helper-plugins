package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	maxLocationsPerMarker = 20
	maxErrorsEmbedded     = 100
	maxValuesEmbedded     = 10000
)

// Finding is one validation or recalculation finding.
type Finding struct {
	Type         string `json:"type"`
	Error        string `json:"error,omitempty"`
	Sheet        string `json:"sheet,omitempty"`
	Cell         string `json:"cell,omitempty"`
	Formula      string `json:"formula,omitempty"`
	Value        string `json:"value,omitempty"`
	Detail       string `json:"detail,omitempty"`
	MissingSheet string `json:"missing_sheet,omitempty"`
	UnknownName  string `json:"unknown_name,omitempty"`
	Heuristic    bool   `json:"heuristic,omitempty"`
}

// Report is the JSON contract shared by validate and recalc.
type Report struct {
	Status          string            `json:"status"`
	File            string            `json:"file"`
	SheetsChecked   []string          `json:"sheets_checked"`
	TotalFormulas   int               `json:"total_formulas"`
	TotalErrors     int               `json:"total_errors"`
	ErrorSummary    map[string]Marker `json:"error_summary"`
	Errors          []Finding         `json:"errors"`
	ErrorsTruncated bool              `json:"errors_truncated"`
	Values          map[string]string `json:"values,omitempty"`
	ValuesTruncated bool              `json:"values_truncated,omitempty"`
	Recalc          map[string]any    `json:"recalc"`
	Scanner         string            `json:"scanner"`
}

// Marker groups findings: count + up to maxLocationsPerMarker locations.
type Marker struct {
	Count     int      `json:"count"`
	Locations []string `json:"locations"`
	Truncated bool     `json:"truncated,omitempty"`
}

// engineVersion reports the Excelize module version from the build info.
func engineVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/xuri/excelize/v2" {
				return "excelize " + dep.Version
			}
		}
	}
	return "excelize (unknown version)"
}

func fail(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, "xlsx: "+msg+"\n", args...)
}

func emitJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		fail("encode output: %v", err)
		os.Exit(2)
	}
}

func openWorkbook(path string) (*excelize.File, error) {
	return excelize.OpenFile(path)
}

// usedBounds returns the reliable used-range bounds of a sheet.
// The <dimension> element is not maintained by all writers (Excelize
// leaves ref="A1" on created files), so the bound is the max of the
// declared dimension and a GetRows scan.
func usedBounds(f *excelize.File, sheet string) (maxCol, maxRow int) {
	if c, r, err := dimensionBounds(f, sheet); err == nil {
		maxCol, maxRow = c, r
	}
	if rows, err := f.GetRows(sheet); err == nil {
		if len(rows) > maxRow {
			maxRow = len(rows)
		}
		for _, row := range rows {
			if len(row) > maxCol {
				maxCol = len(row)
			}
		}
	}
	return maxCol, maxRow
}

// dimensionBounds parses "A1:E2" into max col/row numbers.
func dimensionBounds(f *excelize.File, sheet string) (maxCol, maxRow int, err error) {
	dim, err := f.GetSheetDimension(sheet)
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Split(dim, ":")
	if len(parts) == 1 {
		parts = []string{parts[0], parts[0]}
	}
	_, maxRow, err = excelize.SplitCellName(parts[len(parts)-1])
	if err != nil {
		return 0, 0, err
	}
	colName, _, err := excelize.SplitCellName(parts[len(parts)-1])
	if err != nil {
		return 0, 0, err
	}
	maxCol, err = excelize.ColumnNameToNumber(colName)
	if err != nil {
		return 0, 0, err
	}
	return maxCol, maxRow, nil
}

// FormulaCell is a cell holding a formula.
type FormulaCell struct {
	Sheet   string
	Cell    string
	Formula string
}

// formulaCells enumerates every formula cell on a sheet.
func formulaCells(f *excelize.File, sheet string) ([]FormulaCell, error) {
	maxCol, maxRow := usedBounds(f, sheet)
	var out []FormulaCell
	for r := 1; r <= maxRow; r++ {
		for c := 1; c <= maxCol; c++ {
			name, err := excelize.CoordinatesToCellName(c, r)
			if err != nil {
				continue
			}
			formula, err := f.GetCellFormula(sheet, name)
			if err != nil || formula == "" {
				continue
			}
			out = append(out, FormulaCell{Sheet: sheet, Cell: name, Formula: formula})
		}
	}
	return out, nil
}

// classifyResult maps a CalcCellValue outcome to an error marker.
// ok=false means the formula evaluates to an error.
func classifyResult(value string, err error) (marker, detail string, unsupported, ok bool) {
	if err == nil {
		return "", "", false, true
	}
	detail = err.Error()
	unsupported = strings.Contains(detail, "not support")
	switch {
	case strings.HasPrefix(value, "#"):
		marker = value
	case strings.HasPrefix(detail, "#"):
		marker = detail
	default:
		marker = "#VALUE!"
	}
	return marker, detail, unsupported, false
}

// isErrorMarker reports whether a cached cell value is an Excel error.
func isErrorMarker(v string) bool {
	switch v {
	case "#REF!", "#DIV/0!", "#VALUE!", "#NAME?", "#NULL!", "#NUM!", "#N/A",
		"#GETTING_DATA", "#SPILL!", "#CALC!", "#FIELD!":
		return true
	}
	return false
}

var (
	quotedSheetRe = regexp.MustCompile(`'([^']+)'!`)
	plainSheetRe  = regexp.MustCompile(`(?:^|[^A-Za-z0-9_.$'\p{Han}])([A-Za-z_\p{Han}][A-Za-z0-9_.·\p{Han}]*)!`)
	// nameRefRe captures an identifier plus an optional trailing "(".
	// Callers drop matches whose suffix is "(" (function calls).
	nameRefRe = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]{2,})(\s*\()?`)
	cellRefRe = regexp.MustCompile(`^[A-Z]{1,3}[0-9]+$`)
)

// builtinFuncs is a compact allowlist for the unknown-name heuristic.
var builtinFuncs = map[string]bool{
	"SUM": true, "AVERAGE": true, "COUNT": true, "COUNTA": true,
	"COUNTIF": true, "COUNTIFS": true, "SUMIF": true, "SUMIFS": true,
	"AVERAGEIF": true, "AVERAGEIFS": true, "IF": true, "IFS": true,
	"AND": true, "OR": true, "NOT": true, "XOR": true,
	"VLOOKUP": true, "HLOOKUP": true, "XLOOKUP": true, "LOOKUP": true,
	"MATCH": true, "XMATCH": true, "INDEX": true, "OFFSET": true,
	"INDIRECT": true, "CHOOSE": true, "ROW": true, "ROWS": true,
	"COLUMN": true, "COLUMNS": true, "MIN": true, "MAX": true,
	"MINIFS": true, "MAXIFS": true, "ROUND": true, "ROUNDDOWN": true,
	"ROUNDUP": true, "ABS": true, "POWER": true, "SQRT": true,
	"PRODUCT": true, "MOD": true, "INT": true, "LEN": true,
	"LEFT": true, "RIGHT": true, "MID": true, "TRIM": true,
	"UPPER": true, "LOWER": true, "CONCAT": true, "CONCATENATE": true,
	"TEXTJOIN": true, "TEXT": true, "VALUE": true, "FIND": true,
	"SEARCH": true, "SUBSTITUTE": true, "REPT": true, "EXACT": true,
	"ISBLANK": true, "ISERROR": true, "ISNA": true, "ISNUMBER": true,
	"ISTEXT": true, "IFERROR": true, "IFNA": true, "TRUE": true,
	"FALSE": true, "TODAY": true, "NOW": true, "DATE": true,
	"YEAR": true, "MONTH": true, "DAY": true, "EDATE": true,
	"EOMONTH": true, "PMT": true, "FV": true, "PV": true,
	"NPV": true, "IRR": true, "RATE": true, "NPER": true,
	"LARGE": true, "SMALL": true, "RANK": true, "MEDIAN": true,
	"STDEV": true, "VAR": true, "SUMPRODUCT": true, "SUBTOTAL": true,
	"UNIQUE": true, "SORT": true, "FILTER": true, "SEQUENCE": true,
	"LET": true, "LAMBDA": true, "SWITCH": true, "TEXTSPLIT": true,
	"TEXTAFTER": true, "TEXTBEFORE": true, "WRAPROWS": true,
}

// extractSheetRefs finds sheet names referenced in formula text.
func extractSheetRefs(formula string) []string {
	var refs []string
	for _, m := range quotedSheetRe.FindAllStringSubmatch(formula, -1) {
		refs = append(refs, m[1])
	}
	for _, m := range plainSheetRe.FindAllStringSubmatch(formula, -1) {
		refs = append(refs, m[1])
	}
	return refs
}

// extractNameRefs finds identifiers that look like named-range references.
func extractNameRefs(formula string) []string {
	clean := quotedSheetRe.ReplaceAllString(formula, "")
	clean = plainSheetRe.ReplaceAllString(clean, " ")
	var names []string
	for _, m := range nameRefRe.FindAllStringSubmatch(clean, -1) {
		cand := m[1]
		if m[2] != "" {
			continue // function call, not a name reference
		}
		if cellRefRe.MatchString(cand) {
			continue
		}
		if builtinFuncs[strings.ToUpper(cand)] {
			continue
		}
		names = append(names, cand)
	}
	return names
}

// summarizeErrors groups findings by marker/type with capped locations.
func summarizeErrors(findings []Finding) map[string]Marker {
	summary := map[string]Marker{}
	for _, e := range findings {
		key := e.Type
		if e.Type == "error_value" {
			key = e.Error
		}
		loc := e.Sheet + "!" + e.Cell
		m := summary[key]
		m.Count++
		if len(m.Locations) < maxLocationsPerMarker {
			m.Locations = append(m.Locations, loc)
		} else {
			m.Truncated = true
		}
		summary[key] = m
	}
	if summary == nil {
		summary = map[string]Marker{}
	}
	return summary
}

// buildReport assembles the shared JSON report.
func buildReport(file string, sheets []string, formulas int, findings []Finding,
	values map[string]string, valuesTruncated bool,
	recalc map[string]any, scanner string) Report {
	if findings == nil {
		findings = []Finding{}
	}
	if sheets == nil {
		sheets = []string{}
	}
	embedded := findings
	truncated := false
	if len(findings) > maxErrorsEmbedded {
		embedded = findings[:maxErrorsEmbedded]
		truncated = true
	}
	status := "success"
	if len(findings) > 0 {
		status = "errors_found"
	}
	return Report{
		Status:          status,
		File:            file,
		SheetsChecked:   sheets,
		TotalFormulas:   formulas,
		TotalErrors:     len(findings),
		ErrorSummary:    summarizeErrors(findings),
		Errors:          embedded,
		ErrorsTruncated: truncated,
		Values:          values,
		ValuesTruncated: valuesTruncated,
		Recalc:          recalc,
		Scanner:         scanner,
	}
}
