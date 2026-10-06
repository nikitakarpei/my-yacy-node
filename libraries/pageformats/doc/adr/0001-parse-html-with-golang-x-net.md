# 1. Parse and charset-decode HTML with golang.org/x/net

Date: 2026-07-06

## Status

Accepted

## Context

The crawler must parse fetched HTML to discover links and read meta-robots directives, and it
must decode non-UTF-8 pages before parsing. The standard library has no HTML tree parser or
charset sniffer.

## Decision

We use `golang.org/x/net/html` for parsing and `golang.org/x/net/html/charset` for decoding
(Content-Type, BOM, and meta-charset sniffing), pinned in `go.mod`. `documentextraction` decodes
and parses the fetched body once. Its `htmltree` body holds the parsed tree and gives a copy of
the tree to each derivation that reads html.

## Consequences

The `golang.org/x/net/html` node type is part of the `htmltree` API, thus it reaches each
derivation that reads html. The charset package pulls `golang.org/x/text` for the encoding
tables.
