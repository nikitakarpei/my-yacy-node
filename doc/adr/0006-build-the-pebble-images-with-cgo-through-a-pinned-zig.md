# 6. Build the Pebble images with cgo through a pinned Zig

Date: 2026-10-27

## Status

Accepted

## Context

Without cgo, Pebble keeps its block cache and memtables in the Go heap, and the garbage
collector lets the heap grow to about twice their size. With cgo, Pebble keeps them in C
memory. The images run on `distroless/static`, which has no C library, and their builds
cross-compile for amd64 and arm64.

## Decision

- `tools/tools.lock` pins `zig`, and `tools/install zig` fetches only Zig.
- The images of `yacypeer`, `yacynode`, and `yacycrawler` build their Pebble binary with cgo
  and `zig cc` for the musl target of the image, as a static binary with the `netgo` and
  `osusergo` tags.
- The other images keep `CGO_ENABLED=0`.

## Consequences

- Zig 0.17.0 becomes a pinned build tool. The image build stage installs `xz-utils` to
  unpack it.
- The musl allocator holds the Pebble memory. Go metrics and heap profiles do not show it.
- The Pebble images take longer to build.
