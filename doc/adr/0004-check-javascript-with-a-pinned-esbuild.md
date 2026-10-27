# 4. Check JavaScript with a pinned esbuild

Date: 2026-10-27

## Status

Accepted

## Context

The repository serves hand-written JavaScript in its pages. No gate read it, so a syntax
error or a mistake such as a comparison with `NaN` reached a browser first.

## Decision

- `tools/tools.lock` pins `esbuild`, and `tools/install` fetches its release binary.
- `make lint` runs `esbuild` over every git-tracked `*.js` file that is not minified. A
  syntax error, a feature past ES2022, or a warning fails it.

## Consequences

esbuild 0.28.2 becomes a pinned lint tool. It has no style rules.
