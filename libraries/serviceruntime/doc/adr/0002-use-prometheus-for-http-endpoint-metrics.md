# 2. Use Prometheus for HTTP endpoint metrics

Date: 2026-09-04

## Status

Accepted

## Context

Services use HTTP request counts and duration histograms for operations. The
same metric rule was defined inside individual services.

## Decision

Use `github.com/prometheus/client_golang` in `serviceruntime/httpmetrics`.
The package records a request counter by route and status code, and a duration
histogram by route. It receives one served-request fact from `httpobservation`.

## Consequences

Prometheus becomes a direct dependency of `serviceruntime`. Services use one
tested metric rule and retain their own metric namespaces.
