// Command xlsx is the executable behind the xlsx Helper skill.
//
// Pure-Go spreadsheet control built on Excelize — no Python, no
// LibreOffice. Subcommands:
//
//	xlsx read FILE [--sheet NAME] [--json]        inspect structure + values
//	xlsx validate FILE [--sheet NAME]             static checks (JSON report)
//	xlsx recalc FILE [--output OUT]               evaluate formulas (JSON report)
//	xlsx convert INPUT.csv OUTPUT.xlsx            raw CSV/TSV ingestion
//
// JSON goes to stdout, diagnostics to stderr. Exit codes: 0 = success,
// 1 = errors_found, 2 = operational error.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var code int
	switch os.Args[1] {
	case "read":
		code = cmdRead(os.Args[2:])
	case "validate":
		code = cmdValidate(os.Args[2:])
	case "recalc":
		code = cmdRecalc(os.Args[2:])
	case "convert":
		code = cmdConvert(os.Args[2:])
	case "-h", "--help", "help":
		usage()
	default:
		fail("unknown subcommand %q", os.Args[1])
		usage()
		code = 2
	}
	os.Exit(code)
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: xlsx <read|validate|recalc|convert> [options]

  read FILE [--sheet NAME] [--json]   inspect workbook or CSV/TSV
  validate FILE [--sheet NAME]        static formula checks (JSON)
  recalc FILE [--output OUT]          evaluate formulas, save (JSON)
  convert IN.csv OUT.xlsx             ingest raw values`)
	_ = flag.CommandLine
}
