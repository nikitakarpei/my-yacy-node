# 1. Use Prometheus for Pebble metrics

Date: 2026-10-22

## Status

Accepted

## Context

Operators read the Pebble engine condition, write stalls and write conflicts.
Each service that opens a Pebble engine needs the same metrics.

## Decision

Use `github.com/prometheus/client_golang` in `pebblevault/pebblemetrics`. The
package reads the engine condition and implements the engine observers, and
each service wires it with its own metric namespace.

## Consequences

Prometheus becomes a direct dependency of the pebblevault module. Only code
that imports `pebblemetrics` builds it. Services use one tested metric rule.
