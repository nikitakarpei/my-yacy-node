# 3. Read each page once per stack

Date: 2026-09-02

## Status

Accepted

## Context

Each corpus reads the page in a scrape request. Each corpus can get different
bytes, and each read uses network and origin capacity.

A failed page stays pending in each corpus. Repeated delivery can fill the
intake capacity and stop unrelated pages.

The corpora need the same response bytes, but they derive different values
from those bytes.

## Decision

The pagescrape service reads each requested page once. It offers the page to
all corpora through the scrape page offers stream.

A corpus takes in an offered page. It extracts, derives, and keeps its own
values. It does not read the page from the origin.

The crawler continues to read pages to discover links. A shared proxy cache
can reuse that response for the scrape.

## Consequences

All corpora use the same page bytes for one request.

Adding a corpus does not add an origin read. A corpus can still accept or
reject the offered page independently.

The NATS server must accept messages that contain the maximum offered page.
The supplied stacks use an 8 MB payload limit for pages capped at 2 MB.

