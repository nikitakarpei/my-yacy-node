# 3. Read each page once per stack

Date: 2026-09-02

## Status

Accepted

## Context

When each corpus reads a requested page, the reads can return different
representations and consume origin capacity. The corpora need one input but
derive different values from it.

## Decision

The pagescrape service reads each request once and offers that representation
to all corpora. Each corpus independently accepts or rejects the offer and
derives the values it owns.

Page reads for link discovery remain the crawler's responsibility.

## Consequences

All participating corpora are offered one representation for a scrape request.
Adding a corpus does not add another page read.

The scrape protocol must carry complete page representations, so broker capacity
limits the largest page the stack can offer.
