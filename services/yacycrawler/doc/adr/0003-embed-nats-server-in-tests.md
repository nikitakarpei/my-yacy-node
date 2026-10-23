# 3. Embed a NATS server in broker-edge tests

Date: 2026-07-06

## Status

Accepted. Amended by [17. Serve the crawl broker when no NATS server is configured](0017-serve-the-crawl-broker-when-no-nats-server-is-configured.md)

## Context

The broker edges (`orderintake`, `pagepublication`) are only meaningful against a real
JetStream. Mocking the client would test the mock, not the wire behavior we depend on
(workqueue retention, `DiscardNew` rejection, redelivery).

## Decision

Broker-edge tests run against an in-process `github.com/nats-io/nats-server/v2` instance, a
test-only dependency, matching the approach in `yacycrawlcontract`.

## Consequences

Tests exercise genuine JetStream semantics without external infrastructure. The dependency is
test-scoped and never linked into the service binary.
