# 4. Run PromQL over the metric history

Status: accepted

## Context

The dashboards of the services ask PromQL queries. Without Prometheus, an operator of
`yacypeer` cannot see them. The storage engine of Prometheus links
the SDKs of cloud and Kubernetes service discovery.

`github.com/prometheus/prometheus` is under the Apache 2.0 licence. Its `promql`, `storage`,
and `tsdb/chunkenc` packages hold the query engine and the sample encoding.

## Decision

Link `github.com/prometheus/prometheus` v0.315.1 into `yacypeer`, and use only its query
engine and sample encoding over a history that `yacypeer` keeps itself.

## Consequences

* The dashboards ask the same queries in Grafana and in `yacypeer`.
* `github.com/prometheus/client_golang` must be v1.24.1 or later.
* The binary is larger.
