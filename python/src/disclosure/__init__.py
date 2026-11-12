"""Python wrapper for the CHAOSS ``disclosure`` CLI.

Two ways to use it:

* Command line -- ``disclosure scan ...`` (installed console script) behaves
  exactly like the native binary.
* Library -- ``import disclosure`` and call :func:`scan` / :func:`text`, which
  run the binary and hand back the parsed JSON as a Python ``dict``.

Under the hood this shells out to the prebuilt Go binary; it is not an
in-process binding. That keeps the wrapper tiny and dependency-free.
"""

from __future__ import annotations

import json
import subprocess
from typing import Any, Optional

from ._binary import DISCLOSURE_VERSION, BinaryResolutionError, binary_path

__all__ = [
    "scan",
    "text",
    "version",
    "run",
    "DisclosureError",
    "BinaryResolutionError",
    "__version__",
    "DISCLOSURE_VERSION",
]

__version__ = "0.1.0"

# disclosure CLI exit codes (see the root command's --help):
#   0 = no AI detected, 1 = AI detected, 2 = error.
# Codes 0 and 1 are both successful scans that emit a valid report on stdout;
# only code 2 (and above) signals a real failure.
_EXIT_ERROR = 2


class DisclosureError(RuntimeError):
    """Raised when the disclosure CLI reports an error (exit code >= 2)."""

    def __init__(self, returncode: int, stderr: str) -> None:
        super().__init__(f"disclosure exited with code {returncode}: {stderr.strip()}")
        self.returncode = returncode
        self.stderr = stderr


def run(
    args: list[str],
    *,
    stdin: Optional[str] = None,
) -> subprocess.CompletedProcess:
    """Run the disclosure binary with ``args`` and return the completed process."""
    proc = subprocess.run(
        [str(binary_path()), *args],
        input=stdin,
        capture_output=True,
        text=True,
    )
    if proc.returncode >= _EXIT_ERROR:
        raise DisclosureError(proc.returncode, proc.stderr)
    return proc


def _json(args: list[str], *, stdin: Optional[str] = None) -> Any:
    proc = run([*args, "--format=json"], stdin=stdin)
    return json.loads(proc.stdout)


def scan(
    repo_path: Optional[str] = None,
    *,
    commit_range: Optional[str] = None,
    min_confidence: Optional[str] = None,
    confidence_levels: Optional[str] = None,
) -> Any:
    """Scan a repository's commits for AI signals; return the parsed JSON report.

    ``commit_range`` maps to the CLI ``--range=BASE..HEAD`` flag (named to avoid
    shadowing the ``range`` builtin).
    """
    args = ["scan"]
    if commit_range:
        args.append(f"--range={commit_range}")
    if min_confidence:
        args.append(f"--min-confidence={min_confidence}")
    if confidence_levels:
        args.append(f"--confidence-levels={confidence_levels}")
    if repo_path:
        args.append(repo_path)
    return _json(args)


def text(
    content: Optional[str] = None,
    *,
    input_file: Optional[str] = None,
) -> Any:
    """Scan arbitrary text for AI tool mentions; return the parsed JSON report.

    Pass ``content`` to scan a string (piped via stdin) or ``input_file`` to
    read from a file.
    """
    if content is not None and input_file is not None:
        raise ValueError("pass either content or input_file, not both")
    args = ["text"]
    if input_file is not None:
        args.append(f"--input={input_file}")
        return _json(args)
    return _json(args, stdin=content or "")


def version() -> str:
    """Return the underlying disclosure binary's version string."""
    return run(["version"]).stdout.strip()
