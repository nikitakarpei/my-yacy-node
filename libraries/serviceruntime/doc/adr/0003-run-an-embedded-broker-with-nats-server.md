# 3. Run an embedded broker with nats-server

Date: 2026-10-23

## Status

Accepted

## Context

A service that owns its streams needs an external NATS server to start. An
operator who runs one service binary for a low workload must also install and
run that server.

## Decision

Use `github.com/nats-io/nats-server/v2` in `serviceruntime/jetstreamserver`.
The package starts a NATS server with JetStream in the service process. The
server listens on a configured address and keeps its streams in a configured
store directory. It reports its log lines through `slog`.

The package sets a maximum message size above the NATS default, so that one
message can carry a page. It does not handle process signals, because the
service handles them.

## Consequences

`nats-server` becomes a direct runtime dependency of `serviceruntime`. It was
already in the module graph through `natstestserver`, at the same version as
the server image of the stack.

A service that uses the package links the NATS server into its binary. Other
services connect to that service as they connect to an external NATS server.
When the service stops, its clients lose the broker until it starts again.
