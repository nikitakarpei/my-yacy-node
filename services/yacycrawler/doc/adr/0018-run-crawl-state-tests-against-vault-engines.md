# 18. Run crawl state tests against vault engines

Date: 2026-10-22

## Status

Accepted

## Context

The crawl state stores code against the vault contract. Their behavior needs a fast engine,
and their concurrency needs the engine production runs.

## Decision

Crawl state tests run against the in-memory vault engine
(`libraries/vaultengines/memoryvault`) for behavior, and against the Pebble engine in a
temporary directory for concurrent writes. Both are test-only for the stores.

## Consequences

Tests need no external infrastructure. The e2e suites run the crawler with its state in the
container.
