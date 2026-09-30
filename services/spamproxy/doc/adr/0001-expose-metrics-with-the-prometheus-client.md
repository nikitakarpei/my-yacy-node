# 1. Expose metrics with the Prometheus client

Date: 2026-09-29

## Status

Accepted

## Context

Operators must see how many pages the proxy assesses, how many it relays
without a verdict and why, the spread of scores, and the relays that fail.
The other services serve `/metrics` in the Prometheus format.

## Decision

- `go.mod` pins `github.com/prometheus/client_golang`.
- The service keeps its metrics in a private registry, and serves it on the
  ops address at `/metrics`.
- The relay reports its facts to observers. One observer records metrics,
  and one observer writes the log.

## Consequences

Metrics of all instances aggregate with standard tools. Only the metrics
observer uses the Prometheus client.
