# 10. Derive markdown with JohannesKaufmann/html-to-markdown

Date: 2026-07-14

## Status

Accepted

## Context

Markdown is the first page representation derived from an extracted body rather than produced
by the extractor. Converting the readable article HTML to CommonMark is a solved problem with
no standard-library support, and the conversion runs on arbitrary web pages.

## Decision

We use `github.com/JohannesKaufmann/html-to-markdown/v2` (MIT, pinned in `go.mod`) as the sole
html-to-markdown converter, called through `ConvertString` on serialized article HTML. It is
confined to the `pagemarkdown` derivation.

We pass serialized bytes rather than a parsed tree. The library's open panic on empty text
nodes (issue #197) is reachable only through `ConvertNode` with a caller-mutated tree;
`golang.org/x/net/html` does not produce such nodes when parsing bytes, so the byte-in call
does not reach it.

## Consequences

The converter parses the article HTML a second time, after `htmlpage` has already parsed and
serialized it. Adversarial nesting is bounded by the parser's own element-stack cap from ADR
0005 rather than by this library. Should a future derivation need a tree-level call, issue #197
must be re-examined first.
