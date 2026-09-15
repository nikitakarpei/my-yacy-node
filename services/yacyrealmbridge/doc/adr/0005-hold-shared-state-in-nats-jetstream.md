# 5. Hold shared state in NATS JetStream

Date: 2026-09-14

## Status

Accepted

## Context

One bridge process is a single point of failure for every peer that chose it. Instances behind
one address must agree on which peers are confirmed and on which address each hash leases. The
embedded vault engines lock their directory to one process, so no vault can serve two instances.

## Decision

The bridge holds its views of both realms and its translated address leases in JetStream
key-value buckets it owns, over `github.com/nats-io/nats.go`, and uses no vault. NATS is
required: the bridge does not start without it.

## Consequences

Instances of one bridge are interchangeable, and a NATS cluster gives the bridge its
availability. A lease is one atomic create in a bucket, so two instances never hand one address
to two hashes. Operators run NATS with JetStream next to the bridge. The bridge's addresses in a
realm still sit wherever the realm's technology binds them; NATS does not remove that.
