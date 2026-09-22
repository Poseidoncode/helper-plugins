#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
"""office.pack — repack an XML working directory into an xlsx file.

Thin wrapper over the vendored ``engine/xlsx_pack.py`` (see NOTICE.md).
Keeps the ``scripts/office/`` interface stable while the engine stays frozen.

Usage:
    python scripts/office/pack.py /tmp/work/ output.xlsx
"""

import os
import sys

ENGINE_SCRIPT = os.path.join(
    os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))),
    "engine", "xlsx_pack.py")

if __name__ == "__main__":
    os.execv(sys.executable, [sys.executable, ENGINE_SCRIPT] + sys.argv[1:])
