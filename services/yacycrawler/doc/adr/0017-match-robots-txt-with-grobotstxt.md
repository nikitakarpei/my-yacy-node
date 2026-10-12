# 17. Match robots.txt with grobotstxt

Date: 2026-10-12

## Status

Accepted

## Context

The crawler must obey the robots.txt rules a site states for its product token (RFC 9309)
before it fetches a page. Matching rules to a URL has edge cases in group selection, wildcards
and rule precedence, which a parser of our own would have to get right again.

## Decision

The crawler matches robots.txt with `github.com/jimsmart/grobotstxt` (pinned in `go.mod`), a Go
port of the reference parser Google publishes. It keeps the robots.txt text and lets the
library parse it for each page.

## Consequences

Matching follows the reference parser, including its tolerance for common typos. The library
parses the text again for each page, so a large robots.txt costs CPU on every page of its site.
