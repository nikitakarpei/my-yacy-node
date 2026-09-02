# 5. Own the scrape protocol and outcome feed

Date: 2026-09-02

## Status

Accepted

## Context

Scrape request vocabulary and page vocabulary were in separate library
modules. An external provisioning job created the scrape requests stream.
No one unit owned the complete scrape protocol.

Each corpus reports its own intake result. A caller needs one feed for the
scrape and must not wait for a slow corpus before it sees another result.

## Decision

The pagescrape service owns and creates the scrape requests stream and the
scrape page offers stream. Its contract module owns their names, subjects, and
values. The contract module does not create either stream.

Each corpus publishes an intake receipt on the subject for the page. The
pagescrape service relays each receipt unchanged to that page's scrape outcome
feed as soon as it arrives.

A listener opens the feed before it submits the request. It selects the
receipt for the corpus it needs. The protocol reserves a completed outcome for
a future consumer, but the service does not publish it now.

## Consequences

The service owns the complete scrape without knowing what a corpus stores.
A slow or unavailable corpus does not delay another corpus receipt.

The relay adds one message hop. It gives callers one scrape-owned feed instead
of exposing every corpus protocol.

A future completed outcome can use the consumer count on the page offers
stream. It does not need a configured corpus roster.
