# 13. Serve crawl outcomes over gRPC, not broker request-reply

Date: 2026-08-22

## Status

Accepted

## Context

A caller that orders a crawl and waits for the corpus must know what crawling last did with
a URL: where it resolved to, and whether it was disposed. That state lives in two key-value
buckets the crawler owns. Callers read those buckets directly today, which makes every
reader depend on the crawler's storage layout and on broker credentials.

The crawler already speaks NATS JetStream (ADR 2). Reusing it for this read is possible with
core-NATS request-reply.

## Decision

The crawler serves the read as a gRPC endpoint, `CrawlOutcomes/ReadPage`, at
`YACYCRAWLER_LISTEN_ADDR`. The buckets stay private to the crawler.

Core-NATS request-reply has no durability, no ordering, and no acknowledgment, while every
specification in this repository uses "broker" to mean a channel that has all three. One
word would carry two meanings that the configuration cannot separate. A synchronous read
also must not need broker credentials, and must not fail when durable crawl work fails.

Any crawler instance can answer, because the buckets are shared JetStream key-value. The
endpoint keeps each instance disposable.

## Consequences

Callers depend on a contract, not on a bucket name and a key rule. The crawler listens on a
second port and its callers need a gRPC client. A caller that waits now costs one call per
poll interval instead of two bucket reads.
