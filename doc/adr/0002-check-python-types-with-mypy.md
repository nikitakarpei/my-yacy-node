# 2. Check Python types with mypy

Date: 2026-09-28

## Status

Accepted

## Context

Nothing checked the types of the Python code. Go modules get this check
from the compiler.

## Decision

- The `dev` group of the root `pyproject.toml` pins `mypy`, and `uv.lock`
  locks it.
- `make lint` runs mypy in each Python module with the settings in the root
  `pyproject.toml`.
- Each function must have type annotations.

## Consequences

mypy 2.3.1 becomes a pinned lint tool. A module that imports a package
without type information must name that package in a `tool.mypy.overrides`
entry.
