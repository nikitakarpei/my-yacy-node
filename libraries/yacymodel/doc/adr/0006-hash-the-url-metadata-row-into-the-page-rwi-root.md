# 6. Hash the URL metadata row into the page RWI root

Date: 2026-11-10

## Status

Accepted

## Context

A page attestation signs one value that covers the RWI of a page
([ADR 16](../../../../doc/adr/0016-attest-page-offers-and-rwis-with-hardware-attested-statements.md)).
The URL metadata row holds the postings root, so a hash over the row covers the postings too.

## Decision

The page RWI root is SHA-256 over a domain byte, a format byte and the URL metadata row,
without its `Loaded` day. A row without a postings root has no page RWI root.

The `Loaded` day is left out for two reasons: a crawl of an unchanged page then gives the same
root, and the statement holds the fetch time itself.

## Consequences

- A new field of the row changes the root of each row. Old attestations then fail the check.
- A peer can change the `Loaded` day of an attested row without breaking the attestation.
