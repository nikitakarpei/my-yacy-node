# 5. Format JavaScript and CSS with a pinned Biome

Date: 2026-10-27

## Status

Accepted

## Context

Go and Python code have one format that the gate checks. The JavaScript and CSS of the
pages had none.

## Decision

- `tools/tools.lock` pins `biome`, and `tools/install` fetches its release binary.
- `make fmt` formats every git-tracked `*.js` and `*.css` file that is not minified with
  `biome format`, and `make fmt-check` fails on a file that it would change.
- The root `biome.json` holds the format settings and turns the Biome linter off.

## Consequences

Biome 2.6.3 becomes a pinned format tool.
