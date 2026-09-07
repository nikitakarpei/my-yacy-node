# 4. Hold rankings in NATS when an operator asks

Date: 2026-09-06

## Status

Accepted

## Context

Instances behind one load balancer answer pages of the same query. Each instance holding its
own ranking breaks the agreement between those pages.

## Decision

When an operator supplies a NATS URL, the service holds rankings in a JetStream key-value
bucket it owns, over `github.com/nats-io/nats.go`. The bucket carries the lifetime and the
size limit. Without a URL the service holds rankings in memory.

## Consequences

Instances answer one query from one ranking. NATS is optional: no URL, no broker, no start-up
dependency. A supplied URL that does not answer stops the service from starting.
