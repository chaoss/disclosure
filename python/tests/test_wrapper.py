"""Tests for the disclosure Python wrapper.

These exercise the pure-Python glue (argument construction, JSON parsing, and
exit-code handling) without needing the real Go binary: a tiny shell stub is
dropped into the wrapper's cache directory and pointed at via
``DISCLOSURE_CACHE_DIR``. The stub records the argv it was called with and
emits whatever stdout / exit code the test asks for, mirroring the real CLI's
documented contract (0 = no AI, 1 = AI detected, 2 = error).
"""

from __future__ import annotations

import json
import os
import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

import disclosure  # noqa: E402
from disclosure import _binary  # noqa: E402
from disclosure import __main__ as cli  # noqa: E402

pytestmark = pytest.mark.skipif(
    os.name == "nt", reason="stub binary uses a POSIX shell script"
)


@pytest.fixture
def stub_binary(tmp_path, monkeypatch):
    """Install a fake ``disclosure`` binary and return its argv-log path."""
    cache = tmp_path / "cache"
    version_dir = cache / "disclosure-cli" / _binary.DISCLOSURE_VERSION
    version_dir.mkdir(parents=True)

    argv_log = tmp_path / "argv.txt"
    stub = version_dir / "disclosure"
    stub.write_text(
        "#!/bin/sh\n"
        f'printf "%s\\n" "$@" > "{argv_log}"\n'
        'printf "%s" "$DISCLOSURE_STUB_STDOUT"\n'
        'exit "${DISCLOSURE_STUB_EXIT:-0}"\n'
    )
    stub.chmod(0o755)

    monkeypatch.setenv("DISCLOSURE_CACHE_DIR", str(cache))
    return argv_log


def _set_response(monkeypatch, stdout: str, exit_code: int) -> None:
    monkeypatch.setenv("DISCLOSURE_STUB_STDOUT", stdout)
    monkeypatch.setenv("DISCLOSURE_STUB_EXIT", str(exit_code))


def test_scan_returns_report_when_ai_detected(stub_binary, monkeypatch):
    """Exit code 1 means 'AI detected' — a success, not an error (regression)."""
    payload = {"summary": {"aiCommits": 3}}
    _set_response(monkeypatch, json.dumps(payload), 1)

    result = disclosure.scan("/some/repo", commit_range="a..b", min_confidence="high")

    assert result == payload
    args = stub_binary.read_text().splitlines()
    assert args == [
        "scan",
        "--range=a..b",
        "--min-confidence=high",
        "/some/repo",
        "--format=json",
    ]


def test_text_returns_report_when_ai_detected(stub_binary, monkeypatch):
    _set_response(monkeypatch, json.dumps([{"tool": "Claude"}]), 1)

    result = disclosure.text("I used Claude to write this")

    assert result == [{"tool": "Claude"}]
    assert stub_binary.read_text().splitlines() == ["text", "--format=json"]


def test_scan_returns_report_when_no_ai(stub_binary, monkeypatch):
    _set_response(monkeypatch, json.dumps({"summary": {"aiCommits": 0}}), 0)

    assert disclosure.scan() == {"summary": {"aiCommits": 0}}


def test_error_exit_code_raises(stub_binary, monkeypatch):
    _set_response(monkeypatch, "", 2)

    with pytest.raises(disclosure.DisclosureError) as excinfo:
        disclosure.scan("/bad/repo")

    assert excinfo.value.returncode == 2


def test_text_input_file_uses_input_flag(stub_binary, monkeypatch):
    _set_response(monkeypatch, "[]", 0)

    disclosure.text(input_file="pr-body.txt")

    assert stub_binary.read_text().splitlines() == [
        "text",
        "--input=pr-body.txt",
        "--format=json",
    ]


def test_text_rejects_both_content_and_file(stub_binary):
    with pytest.raises(ValueError):
        disclosure.text("hi", input_file="pr-body.txt")


# --- Binary-resolution / checksum failure handling ---------------------------


def test_main_returns_2_on_binary_resolution_failure(monkeypatch, capsys):
    """A download/cache failure must exit 2 (error), not 1 ('AI detected')."""

    def boom():
        raise _binary.BinaryResolutionError("download failed")

    monkeypatch.setattr(cli, "binary_path", boom)

    assert cli.main() == 2
    assert "download failed" in capsys.readouterr().err


def test_expected_sha256_raises_when_checksums_unavailable(monkeypatch):
    """checksums.txt unreachable => fail closed rather than skip verification."""

    def boom(url):
        raise _binary.BinaryResolutionError("network down")

    monkeypatch.setattr(_binary, "_download", boom)

    with pytest.raises(_binary.BinaryResolutionError):
        _binary._expected_sha256("linux", "amd64")


def test_expected_sha256_raises_when_asset_missing(monkeypatch):
    """checksums.txt present but missing this archive => fail closed."""
    monkeypatch.setattr(
        _binary,
        "_download",
        lambda url: b"deadbeef  disclosure_1.0.0_windows_arm64.tar.gz\n",
    )

    with pytest.raises(_binary.BinaryResolutionError):
        _binary._expected_sha256("linux", "amd64")


def test_fetch_fails_closed_without_checksum(tmp_path, monkeypatch):
    """_fetch must not cache/execute the archive when no checksum is available."""

    def fake_download(url):
        if url.endswith("checksums.txt"):
            raise _binary.BinaryResolutionError("no checksums")
        return b"fake archive bytes"

    monkeypatch.setattr(_binary, "_download", fake_download)

    dest = tmp_path / "disclosure"
    with pytest.raises(_binary.BinaryResolutionError):
        _binary._fetch("linux", "amd64", dest)

    assert not dest.exists()
