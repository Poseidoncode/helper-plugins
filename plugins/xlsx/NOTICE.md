# NOTICE — vendored upstream code

The `engine/` directory is vendored verbatim from the MiniMax xlsx skill:

- Source: https://github.com/MiniMax-AI/skills/tree/main/skills/minimax-xlsx
- Upstream license: **MIT** (Copyright (c) 2026 MiniMax)
- Vendored files: `formula_check.py`, `libreoffice_recalc.py`,
  `shared_strings_builder.py`, `style_audit.py`, `xlsx_add_column.py`,
  `xlsx_insert_row.py`, `xlsx_pack.py`, `xlsx_reader.py`,
  `xlsx_shift_rows.py`, `xlsx_unpack.py`

Per the MIT license terms, the upstream copyright and permission notice
is reproduced below. (The vendored files carry only an
`SPDX-License-Identifier: MIT` header, so this file provides the full
notice MIT requires in all copies.)

```text
MIT License

Copyright (c) 2026 MiniMax

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

Source of the above text: MiniMax-AI/skills `LICENSE` (MIT).

## Update policy

`engine/` is frozen: never edit these files in place. To pick up
upstream fixes, re-download from the source URL above and overwrite
the directory. All Helper-specific behaviour lives in `scripts/`
(the `recalc.py` entry point and the `scripts/office/` thin wrappers),
which treat `engine/` as a read-only dependency loaded via `importlib`
or executed as a subprocess.
