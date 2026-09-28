# 1. Manage Python and its packages with a pinned uv

Date: 2026-09-28

## Status

Accepted

## Context

`make verify` must run only pinned tools. The Python modules used the
`python3` on `PATH` and a `requirements-dev.txt` per module, without hashes
for the packages that the pinned packages pull in. A Python library that two
modules share must resolve to the same versions in both modules.

## Decision

- `tools/tools.lock` pins `uv`, and `tools/install` fetches it like the
  other tools.
- `.python-version` pins the interpreter. uv downloads it into
  `.toolchain/python`.
- The root `pyproject.toml` is a uv workspace. Each Python module is a
  member with its own `pyproject.toml`.
- The root `uv.lock` pins every package and its hash for all members.
  `make verify` syncs one `.venv` from it with `uv sync --locked`.
- `make tidy` runs `uv lock`. `make tidy-check` runs `uv lock --check`.

## Consequences

uv 0.12.19 becomes a pinned build tool. The interpreter and all packages are
the same on each machine and in CI. All members share one version of each
package.
