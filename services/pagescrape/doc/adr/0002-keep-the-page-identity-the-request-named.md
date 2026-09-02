# 2. Keep the page identity the request named

Date: 2026-09-02

## Status

Accepted

## Context

A read can land at a different URL after a redirect. If that URL replaces the
requested identity, a caller cannot wait for and recall the page by the URL it
requested.

## Decision

The page URL in the request remains the identity of the offered page. The URL
where the read lands is a separate value used as the base for document links.

The service keeps no redirection index between these URLs.

## Considered alternatives

Changing the identity to the landed URL is rejected because it breaks recall
and outcome correlation by the requested page URL.

Keeping a redirection index is rejected because it adds state without repairing
a caller that is already waiting on the requested page URL.

## Consequences

A caller requests, waits for, and recalls one page URL across a redirect.

Two requested URLs that land on the same page remain separate identities and
can produce different representations. A scrape stores no second page under
the landed URL.
