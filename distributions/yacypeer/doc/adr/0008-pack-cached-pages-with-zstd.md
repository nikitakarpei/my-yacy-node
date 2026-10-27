# 8. Pack cached pages with zstd

Status: accepted

## Context

A large crawl fetches pages faster than the cache byte maximum holds them for the full
retention, so the sweep evicts pages early. HTML pages pack to about one sixth of their size.

## Decision

Pack each cached page with `github.com/klauspost/compress/zstd` at its default level. The Go
standard library has no zstd encoder, and gzip packs less and runs slower.

## Consequences

* The same byte maximum holds about six times more pages.
* Each kept and each served page costs a pack or an unpack.
* `yacypeer` uses `klauspost/compress` directly, which it already had through Pebble.
