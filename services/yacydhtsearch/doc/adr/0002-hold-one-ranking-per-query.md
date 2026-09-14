# 2. Hold one ranking per query

Date: 2026-09-06

## Status

Accepted

## Context

A page used to cost one fan-out over the DHT. Peers answer under a per-client rate limit, and
the peer set moves between calls, so consecutive pages of one query repeated and skipped
results.

## Decision

One query produces one ranking, up to the record ceiling. The service holds that ranking for a
configured lifetime and cuts every page from it. Peers are asked once per query, not once per
page.

## Consequences

Pages of one query agree with each other while the ranking is held. Results are as old as the
ranking, which the lifetime bounds. Paging deeper than the record ceiling is not possible.
