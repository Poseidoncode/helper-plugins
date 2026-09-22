#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
"""office.unpack — unpack an xlsx into an XML working directory.

Thin wrapper over the vendored ``engine/xlsx_unpack.py`` (see NOTICE.md).
Keeps the ``scripts/office/`` interface stable while the engine stays frozen.

Usage:
    python scripts/office/unpack.py input.xlsx /tmp/work/
"""

import os
import sys

ENGINE_SCRIPT = os.path.join(
    os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__)))),
    "engine", "xlsx_unpack.py")

if __name__ == "__main__":
    os.execv(sys.executable, [sys.executable, ENGINE_SCRIPT] + sys.argv[1:])
