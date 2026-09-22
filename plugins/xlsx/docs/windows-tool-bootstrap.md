# Windows Tool Bootstrap — Go toolchain

This skill needs only a Go toolchain. No Python, no LibreOffice,
no Git Bash.

## 1. Detect

```powershell
go version
```

Need `go1.25` or later (Excelize requirement). Older → §2.

## 2. Install / upgrade

```powershell
winget install GoLang.Go
```

Open a **new** terminal afterwards (PATH changes do not apply to
running shells) and re-run §1.

## 3. Build and use

```powershell
cd plugins\xlsx
go build -o bin\xlsx.exe .\cmd\xlsx
.\bin\xlsx.exe read report.xlsx --sheet Model
.\bin\xlsx.exe recalc report.xlsx
```

`bin/` is gitignored; rebuild after pulling skill updates.
Use forward slashes or backslashes in arguments — both work.
