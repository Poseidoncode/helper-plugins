# NOTICE — dependencies (v2.x, Excelize edition)

As of v2.0.0 this skill contains **no vendored third-party code**.
The `engine/` (MiniMax, MIT) and `scripts/` (Python) trees from v1.x
were removed in the Excelize rewrite.

## Runtime dependency

- `github.com/xuri/excelize/v2` (BSD-3-Clause), pinned in `go.mod` /
  `go.sum`. Consumed as a Go module dependency only — no upstream
  source is copied into this repository.

## History

- v1.x: Python toolchain (pandas + openpyxl + LibreOffice) with a
  vendored MiniMax engine (`NOTICE.md` at tag/commit `b86355e`
  carries the full MIT attribution for that tree).
