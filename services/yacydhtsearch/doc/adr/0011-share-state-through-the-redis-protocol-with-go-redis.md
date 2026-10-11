# 11. Share state through the Redis protocol with go-redis

Date: 2026-10-11

## Status

Accepted

## Context

Instances share rankings, query word document amounts, probe answers and peer presence.
In NATS JetStream key-value buckets, every key is a subject that the broker holds in memory,
so broker memory grows with the queries and words the service meets. This makes the stack
harder to run on hardware with little memory.

## Decision

When an operator supplies a Redis address, the service keeps this state on a server that
speaks the Redis protocol, through `github.com/redis/go-redis/v9` (pinned in `go.mod`): keys
for the caches and the presence, and a stream for the probe answers. Without a Redis address,
each instance keeps its state in memory.

## Consequences

The operator selects the server: Apache Kvrocks keeps state on disk, Redis and Valkey in
memory. An address that does not answer stops the service from starting.
