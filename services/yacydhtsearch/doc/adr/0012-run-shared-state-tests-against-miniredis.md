# 12. Run shared state tests against miniredis

Date: 2026-10-11

## Status

Accepted

## Context

The shared state adapters are only meaningful against a server that speaks the Redis
protocol. Kvrocks and Redis do not embed in a Go test.

## Decision

Shared state tests run against an in-process `github.com/alicebob/miniredis/v2` server
(pinned in `go.mod`), a test-only dependency. Its clock expires keys without real waits.

## Consequences

Tests need no external infrastructure, and the dependency never links into the binary. The
e2e suite runs the service against Kvrocks.
