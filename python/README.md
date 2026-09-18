# disclosure (Python wrapper) — proof of concept

A thin Python wrapper around the [CHAOSS `disclosure`](https://github.com/chaoss/disclosure)
CLI, so it can be installed from PyPI and used from Python.

> **Status: proof of concept.** This wraps the prebuilt `disclosure` binary; it
> is not an in-process binding. Published to **TestPyPI** only for now, to prove
> the approach is useful before deciding on a permanent home and maintenance
> plan (see [issue #82](https://github.com/chaoss/disclosure/issues/82)).

## How it works

On first use the package downloads the prebuilt `disclosure` binary for your
platform from the GitHub release, verifies it against the release
`checksums.txt`, and caches it (under `~/.cache/disclosure-cli/` by default).
After that it just runs the cached binary.

This is the simplest thing that works everywhere from a single wheel. A
production version would instead **bundle the binary in platform-specific
wheels** (per Simon Willison's "Distributing Go binaries" write-up that Andrew
linked on the issue), removing the runtime download.

## Use it as a CLI

```sh
uvx disclosure scan --format=json          # run without installing
# or
pip install disclosure
disclosure scan /path/to/repo
```

## Use it from Python

```python
import disclosure

report = disclosure.scan("/path/to/repo", min_confidence="medium")
print(report)                              # already a dict

findings = disclosure.text("I used Claude to write this")
print(disclosure.version())
```

`scan()` and `text()` return the parsed JSON report as a Python object.

## Install from TestPyPI

```sh
pip install --index-url https://test.pypi.org/simple/ disclosure
```

If the name `disclosure` is already taken on the target index, the package is
published as `disclosure-cli`; then run it with
`uvx --from disclosure-cli disclosure ...`.

## Publishing the PoC (maintainer notes)

```sh
cd python
python -m build
python -m twine upload --repository testpypi dist/*
```

The wrapped binary version is pinned by `DISCLOSURE_VERSION` in
`src/disclosure/_binary.py` (currently `1.0.0`). Override it at runtime with the
`DISCLOSURE_VERSION` env var for local testing.
