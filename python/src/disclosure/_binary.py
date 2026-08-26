"""Locate (and, if needed, download) the platform-native ``disclosure`` binary.

Proof-of-concept strategy: fetch the prebuilt release archive from GitHub on
first use, verify it against the release ``checksums.txt``, and cache the
extracted binary. A production version would instead bundle the binary inside
platform-specific wheels (Simon Willison, "Distributing Go binaries"), which
avoids any download at runtime.
"""

from __future__ import annotations

import hashlib
import os
import platform
import stat
import tarfile
import tempfile
import urllib.request
from pathlib import Path

# The upstream release this PoC wraps. Overridable for local testing.
DISCLOSURE_VERSION = os.environ.get("DISCLOSURE_VERSION", "1.0.0")

_RELEASE_URL = (
    "https://github.com/chaoss/disclosure/releases/download/v{version}/{asset}"
)


class BinaryResolutionError(RuntimeError):
    """Raised when the disclosure binary cannot be located or fetched."""


def _target() -> tuple[str, str]:
    """Return (goos, goarch) matching GoReleaser's asset naming."""
    system = platform.system().lower()
    goos = {"linux": "linux", "darwin": "darwin", "windows": "windows"}.get(system)
    if goos is None:
        raise BinaryResolutionError(f"unsupported OS: {platform.system()!r}")

    machine = platform.machine().lower()
    goarch = {
        "x86_64": "amd64",
        "amd64": "amd64",
        "aarch64": "arm64",
        "arm64": "arm64",
    }.get(machine)
    if goarch is None:
        raise BinaryResolutionError(f"unsupported architecture: {platform.machine()!r}")

    return goos, goarch


def _cache_dir() -> Path:
    override = os.environ.get("DISCLOSURE_CACHE_DIR")
    if override:
        base = Path(override)
    elif platform.system() == "Windows":
        base = Path(os.environ.get("LOCALAPPDATA", Path.home() / "AppData" / "Local"))
    else:
        base = Path(os.environ.get("XDG_CACHE_HOME", Path.home() / ".cache"))
    return base / "disclosure-cli" / DISCLOSURE_VERSION


def _binary_name(goos: str) -> str:
    return "disclosure.exe" if goos == "windows" else "disclosure"


def _download(url: str) -> bytes:
    try:
        with urllib.request.urlopen(url) as resp:  # noqa: S310 (trusted GitHub host)
            return resp.read()
    except Exception as exc:  # pragma: no cover - network failure path
        raise BinaryResolutionError(f"failed to download {url}: {exc}") from exc


def _expected_sha256(goos: str, goarch: str) -> str | None:
    """Look up the archive's checksum from the release checksums.txt."""
    url = _RELEASE_URL.format(
        version=DISCLOSURE_VERSION, asset="checksums.txt"
    )
    archive = f"disclosure_{DISCLOSURE_VERSION}_{goos}_{goarch}.tar.gz"
    try:
        text = _download(url).decode("utf-8")
    except BinaryResolutionError:
        return None  # checksums are best-effort; don't block the PoC if absent
    for line in text.splitlines():
        parts = line.split()
        if len(parts) == 2 and parts[1] == archive:
            return parts[0]
    return None


def _fetch(goos: str, goarch: str, dest: Path) -> None:
    archive = f"disclosure_{DISCLOSURE_VERSION}_{goos}_{goarch}.tar.gz"
    url = _RELEASE_URL.format(version=DISCLOSURE_VERSION, asset=archive)
    blob = _download(url)

    expected = _expected_sha256(goos, goarch)
    if expected is not None:
        actual = hashlib.sha256(blob).hexdigest()
        if actual != expected:
            raise BinaryResolutionError(
                f"checksum mismatch for {archive}: expected {expected}, got {actual}"
            )

    name = _binary_name(goos)
    dest.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory() as tmp:
        tar_path = Path(tmp) / archive
        tar_path.write_bytes(blob)
        with tarfile.open(tar_path) as tf:
            member = tf.getmember(name)
            extracted = tf.extractfile(member)
            if extracted is None:
                raise BinaryResolutionError(f"{name} not found in {archive}")
            dest.write_bytes(extracted.read())
    dest.chmod(dest.stat().st_mode | stat.S_IEXEC | stat.S_IXGRP | stat.S_IXOTH)


def binary_path() -> Path:
    """Return the path to a runnable disclosure binary, fetching it if needed.

    An already-installed ``disclosure`` on ``PATH`` is not used: pinning to a
    known release keeps the wrapper's behaviour reproducible.
    """
    goos, goarch = _target()
    path = _cache_dir() / _binary_name(goos)
    if not path.exists():
        _fetch(goos, goarch, path)
    return path
