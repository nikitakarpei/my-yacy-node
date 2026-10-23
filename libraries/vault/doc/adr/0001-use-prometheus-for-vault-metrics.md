# 1. Use Prometheus for vault metrics

Date: 2026-10-22

## Status

Accepted

## Context

Vault removes expired records in the background, and operators need to see
that removal work. Each service defined its own metrics for the same vault work.

## Decision

Use `github.com/prometheus/client_golang` in `vault/vaultmetrics`. The package
implements the vault observers, and each service wires it with its own metric
namespace.

## Consequences

Prometheus becomes a direct dependency of the vault module. Only code that
imports `vaultmetrics` builds it. Services use one tested metric rule.
