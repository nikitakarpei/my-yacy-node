# 6. Serve the scrape broker when no NATS server is configured

Date: 2026-10-23

## Status

Accepted

## Context

The service owns the scrape request stream, the page offer stream, and the
outcome feed. Each corpus and each caller reaches the pipeline through a NATS
server. An operator who runs the service for a low workload must also run that
server. A bundle of services in one process needs a broker it can start.

## Decision

When `SCRAPE_NATS_URL` is unset, the service starts a NATS server with
JetStream through `serviceruntime/jetstreamserver` and connects to it. The
server listens on `PAGESCRAPE_NATS_ADDR` and keeps its streams in
`PAGESCRAPE_NATS_STORE_DIR`. Other services set their NATS URL to that
address.

When `SCRAPE_NATS_URL` is set, the service connects to that server and serves
none.

## Consequences

One binary runs the scrape pipeline without other infrastructure. Corpora,
callers, and the service itself use the same subjects and streams on either
server.

While the service is stopped, its clients have no broker. A deployment that
must keep the broker through a service restart sets `SCRAPE_NATS_URL`.
