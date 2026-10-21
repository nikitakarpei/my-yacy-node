# 15. Keep crawl state behind the Redis protocol with go-redis

Date: 2026-10-11

## Status

Superseded by [17. Keep crawl state in a vault on Pebble](0017-keep-crawl-state-in-a-vault-on-pebble.md)

## Context

The crawler keeps state for each order and each URL it meets. In NATS JetStream key-value
buckets, every key is a subject that the broker holds in memory, so broker memory grows with
the crawl. This makes the stack harder to run on hardware with little memory. Crawler
instances must share this state, so an embedded store does not fit.

## Decision

The crawler keeps accepted orders and page visit state on a server that speaks the Redis
protocol, through `github.com/redis/go-redis/v9` (pinned in `go.mod`). It sends only commands
that Redis and Apache Kvrocks both serve, and changes a record with a single-key
compare-and-set script. JetStream keeps only streams.

## Consequences

The operator selects the server: Kvrocks keeps keys on disk with fixed memory, Redis and
Valkey keep them in memory. The crawler does not start without the server. Single-key
records spread evenly over the shards of a cluster.
