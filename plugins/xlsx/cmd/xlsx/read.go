package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

// SheetView is the JSON view of one sheet (or CSV input).
type SheetView struct {
	Name      string     `json:"name"`
	Dimension string     `json:"dimension,omitempty"`
	Rows      int        `json:"rows"`
	Cols      int        `json:"cols"`
	Headers   []string   `json:"headers"`
	Preview   [][]string `json:"preview"`
}

// ReadOutput is the machine-readable read report.
type ReadOutput struct {
	File   string      `json:"file"`
	Sheets []SheetView `json:"sheets"`
}

func cmdRead(args []string) int {
	fs := flag.NewFlagSet("read", flag.ContinueOnError)
	sheetFilter := fs.String("sheet", "", "analyze a specific sheet only")
	asJSON := fs.Bool("json", false, "machine-readable JSON output")
	preview := fs.Int("preview", 10, "data rows previewed per sheet")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() < 1 {
		fail("usage: xlsx read FILE [--sheet NAME] [--json]")
		return 2
	}
	path := fs.Arg(0)
	var out ReadOutput
	out.File = path

	if isCSV(path) {
		view, err := readCSVView(path, *preview)
		if err != nil {
			fail("%v", err)
			return 2
		}
		out.Sheets = []SheetView{view}
	} else {
		f, err := openWorkbook(path)
		if err != nil {
			fail("open %s: %v", path, err)
			return 2
		}
		defer f.Close()
		for _, name := range f.GetSheetList() {
			if *sheetFilter != "" && name != *sheetFilter {
				continue
			}
			view, err := readSheetView(f, name, *preview)
			if err != nil {
				fail("read sheet %s: %v", name, err)
				return 2
			}
			out.Sheets = append(out.Sheets, view)
		}
	}

	if *asJSON {
		emitJSON(out)
		return 0
	}
	printReadText(out)
	return 0
}

func printReadText(out ReadOutput) {
	fmt.Printf("File: %s (%d sheet(s))\n", out.File, len(out.Sheets))
	for _, s := range out.Sheets {
		fmt.Printf("\n[%s] %d rows x %d cols", s.Name, s.Rows, s.Cols)
		if s.Dimension != "" {
			fmt.Printf(" (%s)", s.Dimension)
		}
		fmt.Println()
		fmt.Printf("  headers: %s\n", strings.Join(s.Headers, " | "))
		for _, row := range s.Preview {
			fmt.Printf("  %s\n", strings.Join(row, " | "))
		}
	}
}

func readSheetView(f *excelize.File, name string, preview int) (SheetView, error) {
	view := SheetView{Name: name, Headers: []string{}}
	rows, err := f.GetRows(name)
	if err != nil {
		return view, err
	}
	// usedBounds, not the declared <dimension> (not maintained by all writers).
	if mc, mr := usedBounds(f, name); mc > 0 && mr > 0 {
		tl, _ := excelize.CoordinatesToCellName(1, 1)
		br, berr := excelize.CoordinatesToCellName(mc, mr)
		if berr == nil {
			view.Dimension = tl + ":" + br
		}
	}
	view.Rows = len(rows)
	for _, r := range rows {
		if len(r) > view.Cols {
			view.Cols = len(r)
		}
	}
	if len(rows) > 0 {
		view.Headers = append([]string{}, rows[0]...)
		rows = rows[1:]
	}
	for i, r := range rows {
		if i >= preview {
			break
		}
		view.Preview = append(view.Preview, append([]string{}, r...))
	}
	if view.Preview == nil {
		view.Preview = [][]string{}
	}
	return view, nil
}

func isCSV(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".csv") || strings.HasSuffix(lower, ".tsv")
}

func readCSVView(path string, preview int) (SheetView, error) {
	fh, err := os.Open(path)
	if err != nil {
		return SheetView{}, err
	}
	defer fh.Close()
	r := csv.NewReader(fh)
	r.FieldsPerRecord = -1
	if strings.HasSuffix(strings.ToLower(path), ".tsv") {
		r.Comma = '\t'
	}
	records, err := r.ReadAll()
	if err != nil {
		return SheetView{}, err
	}
	view := SheetView{Name: "data", Headers: []string{}, Preview: [][]string{}}
	if len(records) > 0 {
		view.Headers = append([]string{}, records[0]...)
		records = records[1:]
	}
	view.Rows = len(records)
	for _, rec := range records {
		if len(rec) > view.Cols {
			view.Cols = len(rec)
		}
	}
	for i, rec := range records {
		if i >= preview {
			break
		}
		view.Preview = append(view.Preview, append([]string{}, rec...))
	}
	return view, nil
}
