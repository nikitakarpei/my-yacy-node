# 2. Expose metrics with the Prometheus client

Date: 2026-10-10

## Status

Accepted

## Context

Operators must see how many pages the proxy fetches, by status, and the
fetches and relays that fail. The other services serve `/metrics` in the
Prometheus format.

## Decision

- `go.mod` pins `github.com/prometheus/client_golang`.
- The service keeps its metrics in a private registry, and serves it on the
  ops address at `/metrics`.
- Each unit reports its facts to observers. One observer records metrics, and
  one observer writes the log.

## Consequences

Metrics of all instances aggregate with standard tools. Only the metrics
observers use the Prometheus client.
