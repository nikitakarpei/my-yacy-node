# 5. Own the scrape protocol and outcome feed

Date: 2026-09-02

## Status

Accepted

## Context

Scrape requests, page offers, and corpus receipts form one pipeline. Separate
protocol modules and external stream provisioning left no service responsible
for that pipeline as a whole.

Callers also need scrape outcomes without depending on each corpus protocol.

## Decision

The pagescrape service owns the scrape protocol, its streams, and the outcome
feed. Its contract module defines protocol vocabulary but provisions no broker
resources.

Each corpus reports its own outcome. The service carries each report to the
page's scrape outcome feed without waiting for other corpora.

## Consequences

One service owns the pipeline while each corpus remains responsible for its own
content and acceptance rules.

A slow or unavailable corpus does not delay another corpus outcome. The outcome
feed adds a message hop. Outcomes are unavailable to listeners that subscribe
too late.
