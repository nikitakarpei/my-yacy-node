# 4. Defer a scrape by scheduling its request

Date: 2026-09-02

## Status

Accepted

## Context

A page can ask a reader to wait before another request. Broker negative
acknowledgement keeps the request pending and gives no total time limit.

Some callers need an immediate result and cannot wait for a deferred request.

## Decision

The service republishes a deferred request with a delivery time and
acknowledges the current message. The scrape requests stream holds scheduled
messages and delivers them to the scrape request subject at that time.

The first deferred request records when deferral started. The service reports
a `deferred-too-long` failure when the configured deferral window is spent.

A caller can request no deferral. The service then reports a `deferred`
failure and does not schedule another message.

## Consequences

A deferred page does not occupy intake capacity while it waits.

The deferral window gives every request a final outcome. A caller that gives
up on deferral also gets a final outcome on the first response.

The stack requires NATS 2.14 or later for scheduled messages.
