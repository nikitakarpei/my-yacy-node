# 1. Read pages with golang.org/x/net

Date: 2026-09-29

## Status

Accepted

## Context

The spam model reads the title, visible text and links of a page, and the
registrable domain of each address. Many pages are not UTF-8. The standard
library has no HTML tree parser, no charset sniffer and no public suffix list.

## Decision

- `go.mod` pins `golang.org/x/net` and `golang.org/x/text`.
- `htmlreading` parses a page once with `golang.org/x/net/html`, and decodes
  it with `golang.org/x/net/html/charset`.
- `pagesite` finds the registrable domain with `golang.org/x/net/publicsuffix`.

## Consequences

The library needs no cgo. The public suffix list changes only with a new
version of `golang.org/x/net`.
