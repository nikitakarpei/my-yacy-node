# 17. Serve the crawl broker when no NATS server is configured

Date: 2026-10-23

## Status

Accepted. Amends [3](0003-embed-nats-server-in-tests.md).

## Context

The crawler owns the order stream and the crawled page stream. Order sources
and page consumers reach them through a NATS server. An operator who runs the
crawler for a low workload must also run that server. A bundle of services in
one process needs a broker it can start.

## Decision

When `CRAWL_NATS_URL` is unset, the crawler starts a NATS server with
JetStream through `serviceruntime/jetstreamserver` and connects to it. The
server listens on `YACYCRAWLER_NATS_ADDR` and keeps its streams in
`YACYCRAWLER_NATS_STORE_DIR`. Order sources and page consumers set their NATS
URL to that address.

When `CRAWL_NATS_URL` is set, the crawler connects to that server and serves
none.

## Consequences

One binary runs the crawler without a separate broker. `nats-server` is now
linked into the service binary, so decision 3 no longer keeps it test-scoped.

While the crawler is stopped, its clients have no broker. Instances that each
serve their own server share no order stream, so several instances set
`CRAWL_NATS_URL` to one server.
