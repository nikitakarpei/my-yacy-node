# 1. Attest RWIs

Date: 2026-11-10

## Status

Accepted

## Context

The page attestation of an RWI covers each step of the pipeline
([ADR 16](../../../../doc/adr/0016-attest-page-offers-and-rwis-with-hardware-attested-statements.md)).
pagerwi turns an attested page offer into the RWI of the page.

## Decision

pagerwi signs a `page-rwi` statement. Its subject is the page RWI root. It also holds the page
URL, the landed URL when it is not the page URL, and the drand round of the response
attestation. One RFC 3161 time-stamp covers it.

pagerwi attests a page only when the fetch URL is the page URL, and the page URL is https on
the default port. Every other page gets an RWI without attestation.

pagerwi takes the `Loaded` day from the fetch time that the page offer carries.

## Consequences

- Pages without postings and pages with an http page URL get no attestation.
- The `Loaded` day of an attested row is not covered: a peer that passes the row on can change
  it. The fetch time comes from the check, not from the row.
