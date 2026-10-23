# 1. Use Prometheus for vault metrics

Date: 2026-10-22

## Status

Accepted

## Context

Operators read vault transactions, capacity, records per bucket, member sets
and the removal of expired records. Each service that opens a vault needs the
same metrics.

## Decision

Use `github.com/prometheus/client_golang` in `vault/vaultmetrics`. The package
implements the vault observers and reads vault capacity and record counts, and
each service wires it with its own metric namespace.

## Consequences

Prometheus becomes a direct dependency of the vault module. Only code that
imports `vaultmetrics` builds it. Services use one tested metric rule.
