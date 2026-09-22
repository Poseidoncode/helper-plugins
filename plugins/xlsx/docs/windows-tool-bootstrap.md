# Windows Tool Bootstrap — Git Bash detection and install

All scripts and examples in this skill use bash syntax. On Windows,
execute them via **Git Bash**.

## 1. Detect

```powershell
where bash
where python
```

- `bash` found → you have Git Bash; run everything as
  `bash -c "python scripts/recalc.py file.xlsx 60"`.
- `bash` missing → install Git for Windows ( §2 ), which bundles it.
- `python` missing → install Python 3.9+ and re-open the terminal.

## 2. Install

| Tool | Command |
|---|---|
| Git + Git Bash | `winget install Git.Git` |
| Python 3.9+ | `winget install Python.Python.3` |
| LibreOffice (for `recalc.py`) | `winget install TheDocumentFoundation.LibreOffice` |

After installing, open a **new** Git Bash window (PATH changes do
not apply to running shells) and re-run §1.

## 3. LibreOffice notes

- `scripts/recalc.py` uses headless `--convert-to` (no macro
  installation); on Windows it finds `soffice.exe` via `PATH` or the
  default install location. If detection fails, add LibreOffice's
  `program` directory to `PATH` manually.
- No-LibreOffice fallback: open the workbook in the LibreOffice GUI,
  press Ctrl+Shift+F9 (recalculate all), save, then run
  `python scripts/recalc.py file.xlsx --static-only` to verify.

## 4. Path mapping

Git Bash maps `/tmp/` automatically; all skill scripts work as-is.
Use forward slashes in arguments even on Windows.
