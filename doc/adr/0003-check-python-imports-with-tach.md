# 3. Check Python imports with Tach

Date: 2026-09-29

## Status

Accepted

## Context

`make arch` checks the imports of each Go module with go-arch-lint. Each Go
component lists the components it may use. Python modules had no equivalent
check.

## Decision

- The `dev` group of the root `pyproject.toml` pins `tach`, and `uv.lock`
  locks it.
- Each Python module has a `tach.toml`. Each of its modules lists the
  modules it may use in `depends_on`, and the third-party packages it may
  use in `depends_on_external`.
- A package that the host provides at runtime goes in `external.exclude`.
- `make arch` runs `tach check` and `tach check-external` in each Python
  module.
- Nobody runs `tach show` or `tach upload`. These commands send data to the
  Gauge web service.

## Consequences

Tach 0.35.1 becomes a pinned lint tool. A Python module without a
`tach.toml` fails `make arch`. An import that a module does not list fails
`make arch`.
