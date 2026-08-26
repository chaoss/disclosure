"""Console entry point: forward all arguments to the disclosure binary.

``disclosure <args>`` and ``python -m disclosure <args>`` both behave exactly
like invoking the native CLI, including stdin/stdout piping and exit codes.
"""

from __future__ import annotations

import subprocess
import sys

from ._binary import BinaryResolutionError, binary_path


def main() -> int:
    try:
        binary = binary_path()
    except BinaryResolutionError as exc:
        print(f"disclosure: {exc}", file=sys.stderr)
        return 1
    # Inherit stdio so pipes (`... | disclosure text`) and TTY output work.
    return subprocess.run([str(binary), *sys.argv[1:]]).returncode


if __name__ == "__main__":
    sys.exit(main())
