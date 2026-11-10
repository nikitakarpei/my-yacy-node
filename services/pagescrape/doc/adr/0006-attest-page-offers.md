# 6. Attest page offers

Date: 2026-11-10

## Status

Accepted

## Context

The page attestation of an RWI covers each step of the pipeline
([ADR 16](../../../../doc/adr/0016-attest-page-offers-and-rwis-with-hardware-attested-statements.md)).
pagescrape turns the response attestations of a fetch into a page offer.

## Decision

pagescrape signs a `page-offer` statement. Its subject is the SHA-256 of the page offer message
body as sent. It also holds the fetch URL and the drand round of the response attestation of the
last hop.

pagescrape sends the bundle in a header of the page offer message, beside the body, so the
digest covers the body exactly as sent.

The page offer carries the fetch time. The fetch time is the time of the drand round of the
response attestation of the last hop. Without a response attestation, it is the clock of
pagescrape.

## Consequences

- A page offer is attested only when each hop has a response attestation of the same key.
- Offers signed by an earlier key of the app get no attestation.
- A change of the offer body after signing breaks the attestation.
