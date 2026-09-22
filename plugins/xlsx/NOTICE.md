# NOTICE — vendored upstream code

The `engine/` directory is vendored verbatim from the MiniMax xlsx skill:

- Source: https://github.com/MiniMax-AI/skills/tree/main/skills/minimax-xlsx
- Upstream license: **MIT** (Copyright (c) 2026 MiniMax)
- Vendored files: `formula_check.py`, `libreoffice_recalc.py`,
  `shared_strings_builder.py`, `style_audit.py`, `xlsx_add_column.py`,
  `xlsx_insert_row.py`, `xlsx_pack.py`, `xlsx_reader.py`,
  `xlsx_shift_rows.py`, `xlsx_unpack.py`

Per the MIT license terms, the upstream copyright notice is preserved
in each file header (`SPDX-License-Identifier: MIT`).

## Update policy

`engine/` is frozen: never edit these files in place. To pick up
upstream fixes, re-download from the source URL above and overwrite
the directory. All Helper-specific behaviour lives in `scripts/`
(the `recalc.py` entry point and the `scripts/office/` thin wrappers),
which treat `engine/` as a read-only dependency loaded via `importlib`
or executed as a subprocess.
