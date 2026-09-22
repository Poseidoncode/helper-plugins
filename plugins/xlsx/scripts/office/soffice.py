#!/usr/bin/env python3
# SPDX-License-Identifier: MIT
"""office.soffice — hardened LibreOffice discovery and execution helper.

This module is the single place where this skill touches LibreOffice:

- :func:`find_soffice` / :func:`libreoffice_version` — locate the binary
  (reuses the vendored ``engine.libreoffice_recalc`` implementation so
  discovery behaviour never drifts between entry points).
- :func:`get_soffice_env` — build the subprocess environment used for
  every headless invocation. It augments ``PATH`` (macOS bundle location)
  and sets headless-friendly variables. ``scripts/recalc.py`` applies it
  transparently before calling into the engine.
- :func:`run_convert` — run one ``--headless --convert-to`` call with a
  bounded timeout and structured errors.

Sandbox note (macOS App Sandbox / Linux seccomp): if the host denies
``AF_UNIX`` sockets, LibreOffice fails regardless of environment
variables — no ``LD_PRELOAD`` shim is bundled here. That failure
surfaces as a ``SofficeError`` with an actionable message instead of a
hang. Ensure a C toolchain (``gcc``) is *not* required: there is nothing
to compile.

CLI:
    python scripts/office/soffice.py --check
    python scripts/office/soffice.py --convert-to xlsx in.xlsx --outdir /tmp/out
"""

import argparse
import importlib.util
import os
import shutil
import subprocess
import sys


def _engine_path() -> str:
    here = os.path.dirname(os.path.abspath(__file__))
    return os.path.join(os.path.dirname(here), "..", "engine")


def _load_engine_module(name: str):
    path = os.path.join(_engine_path(), name + ".py")
    spec = importlib.util.spec_from_file_location("xlsx_engine_" + name, path)
    if spec is None or spec.loader is None:
        raise SofficeError("cannot load engine module: " + path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class SofficeError(RuntimeError):
    """Raised when LibreOffice cannot be located or executed."""


def find_soffice() -> str | None:
    """Locate the soffice binary, or return None. Never raises."""
    try:
        engine = _load_engine_module("libreoffice_recalc")
        return engine.find_soffice()
    except Exception:
        return None


def libreoffice_version(soffice: str) -> str:
    """Return the LibreOffice version string, or 'unknown'."""
    try:
        engine = _load_engine_module("libreoffice_recalc")
        return engine.get_libreoffice_version(soffice)
    except Exception:
        return "unknown"


def get_soffice_env(base: dict | None = None) -> dict:
    """Build the environment for headless LibreOffice invocations.

    - Copies ``base`` (defaults to ``os.environ``) so callers never mutate
      the process environment by accident.
    - Prepends the macOS application-bundle directory to ``PATH`` so a
      default ``brew install --cask libreoffice`` install is found even
      when ``soffice`` is not on ``PATH``.
    - Sets ``SAL_USE_VCLPLUGIN=gen`` (generic backend; avoids GUI toolkit
      probing in headless mode).
    """
    env = dict(os.environ if base is None else base)
    mac_bundle = "/Applications/LibreOffice.app/Contents/MacOS"
    if os.path.isdir(mac_bundle):
        path = env.get("PATH", "")
        if mac_bundle not in path.split(os.pathsep):
            env["PATH"] = mac_bundle + os.pathsep + path
    env.setdefault("SAL_USE_VCLPLUGIN", "gen")
    return env


def run_convert(
    input_path: str,
    outdir: str,
    convert_to: str = "xlsx",
    timeout: int = 60,
    extra_args: list | None = None,
) -> subprocess.CompletedProcess:
    """Run ``soffice --headless --convert-to ...`` with a bounded timeout.

    Raises:
        SofficeError: binary missing, timeout, or non-zero exit.
    """
    soffice = find_soffice()
    if not soffice:
        raise SofficeError(
            "LibreOffice not found. Install it to enable dynamic recalculation.\n"
            "  macOS:  brew install --cask libreoffice\n"
            "  Linux:  sudo apt-get install -y libreoffice\n"
            "  Windows: winget install TheDocumentFoundation.LibreOffice"
        )
    cmd = [
        soffice,
        "--headless",
        "--norestore",
        "--convert-to",
        convert_to,
        "--outdir",
        outdir,
    ]
    cmd.extend(extra_args or [])
    cmd.append(input_path)
    try:
        result = subprocess.run(
            cmd, capture_output=True, timeout=timeout, env=get_soffice_env()
        )
    except subprocess.TimeoutExpired as e:
        raise SofficeError(
            "soffice timed out after %ds. Raise the timeout "
            "(python scripts/recalc.py file.xlsx 180) or simplify the file. "
            "See docs/recalc-guide.md §5." % timeout
        ) from e
    except OSError as e:
        raise SofficeError("cannot execute %s: %s" % (soffice, e)) from e
    if result.returncode != 0:
        stderr = result.stderr.decode(errors="replace").strip()
        if "Address already in use" in stderr or "AF_UNIX" in stderr:
            raise SofficeError(
                "LibreOffice socket setup failed (AF_UNIX denied by sandbox). "
                "Retry outside the sandbox or run recalc.py on an unrestricted "
                "host. See docs/recalc-guide.md §6."
            )
        raise SofficeError(
            "soffice exited with code %d.\nstderr: %s"
            % (result.returncode, stderr[-2000:])
        )
    return result


def main(argv: list | None = None) -> int:
    parser = argparse.ArgumentParser(description="Hardened LibreOffice runner.")
    parser.add_argument("--check", action="store_true",
                        help="Only check availability, then exit.")
    parser.add_argument("--convert-to", default="xlsx",
                        help="Target format for --convert-to (default: xlsx).")
    parser.add_argument("input", nargs="?",
                        help="Input file for conversion.")
    parser.add_argument("--outdir", default=".",
                        help="Output directory for conversion.")
    parser.add_argument("--timeout", type=int, default=60,
                        help="Timeout in seconds (default: 60).")
    args = parser.parse_args(argv)

    if args.check:
        soffice = find_soffice()
        if soffice:
            print("LibreOffice available: " + soffice)
            print("Version: " + libreoffice_version(soffice))
            return 0
        print("LibreOffice NOT available.")
        return 2

    if not args.input:
        parser.print_help()
        return 1
    os.makedirs(args.outdir, exist_ok=True)
    try:
        result = run_convert(args.input, args.outdir,
                             convert_to=args.convert_to,
                             timeout=args.timeout)
    except SofficeError as e:
        print("ERROR: %s" % e, file=sys.stderr)
        return 1
    print(result.stdout.decode(errors="replace").strip())
    return 0


if __name__ == "__main__":
    sys.exit(main())
