# 11. Share state through the Redis protocol with go-redis

Date: 2026-10-11

## Status

Accepted

## Context

Instances share rankings, query word document amounts, probe answers and peer presence.
NATS JetStream carries streams only, and the service should depend on one shared server.

## Decision

When an operator supplies a state server, the service keeps this state on a server that
speaks the Redis protocol, through `github.com/redis/go-redis/v9` (pinned in `go.mod`): keys
for the caches and the presence, and a stream for the probe answers. Without a state server,
each instance keeps its state in memory.

## Consequences

The operator selects the server: Apache Kvrocks keeps state on disk, Redis and Valkey in
memory. A state server that does not answer stops the service from starting.
