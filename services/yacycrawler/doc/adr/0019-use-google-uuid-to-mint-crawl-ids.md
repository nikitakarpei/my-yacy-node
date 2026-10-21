# 19. Use google/uuid to mint crawl IDs

Date: 2026-10-22

## Status

Accepted

## Context

The crawler names each crawl it runs for an accepted order. Crawl state records key on that
name, so it must be short, unique without coordination, and ordered by the time the crawl
began.

## Decision

The crawler mints each crawl ID as a version 7 UUID with `github.com/google/uuid` (pinned in
`go.mod`), the version visitcrawl already pins.

## Consequences

A crawl ID takes 16 bytes in a record key and sorts by the time its crawl began. The crawl
ID package depends on the library; the rest of the crawler sees only the crawl ID type.
