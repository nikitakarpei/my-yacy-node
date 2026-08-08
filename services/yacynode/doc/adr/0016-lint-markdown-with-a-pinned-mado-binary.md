# 16. Lint Markdown with a pinned `mado` binary

Date: 2026-08-09

## Status

Accepted

## Context

Markdown docs (specifications, ADRs, configuration references) have no automated
check. Structural drift — broken lists, missing code-fence languages, bare URLs —
can land unnoticed.

## Decision

`make lint-md` runs `mado` over every git-tracked `*.md` file and feeds `make
verify` through `lint`. `mado` is a single static Rust binary with a
markdownlint-compatible rule set, fetched and checksum-verified through
`tools/tools.lock`, matching `golangci-lint` and `ruff`.

`mado.toml` at the repo root disables MD013 (line-length): this project's docs
do not hard-wrap prose. `AGENTS.md` and `CLAUDE.md` are excluded, since they
are headerless directive files by design.

## Consequences

`mado` v0.3.1 becomes a build-time dependency; this ADR is its dependency
record.
