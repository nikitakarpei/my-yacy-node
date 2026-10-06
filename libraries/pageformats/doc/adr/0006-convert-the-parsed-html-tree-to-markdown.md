# 6. Convert the parsed html tree to markdown

Date: 2026-10-06

## Status

Accepted

Supersedes [4. Derive markdown with JohannesKaufmann/html-to-markdown](0004-derive-markdown-with-johanneskaufmann-html-to-markdown.md).
Keeps the converter that ADR 4 selects.

## Context

A document keeps its html body as the tree that extraction parsed. Each derivation that reads
html gets its own copy of that tree. A converter that reads bytes makes the body render the tree
and then parses the bytes again. On 6653 stored pages, this render and parse take approximately
20 ms of the 67 ms that markdown from the whole document takes.

ADR 4 passes bytes because of issue #197: the converter panics on an empty text node in a tree
that the caller changed. Readability changes the tree that it reads, thus the readable html is
such a tree.

## Decision

We use `github.com/JohannesKaufmann/html-to-markdown/v2` (MIT, pinned in `go.mod`) as the only
html-to-markdown converter. It is confined to the `markdown` derivations. Each derivation calls
`ConvertNode` on its copy of the tree. `ConvertNode` changes the tree that it reads, thus it
never gets the tree that the document holds.

We examined issue #197 again on v2.5.2. The collapse step removes a text node that becomes
empty. Trees from readability and trees with empty text nodes do not cause a panic.

## Consequences

The converter does not parse the html again. Markdown can differ from the markdown of the
rendered html when the parser reads the rendered html differently from the fetched body.

Adversarial nesting stays bounded, because the parser from ADR 1 rejects html that is nested
deeper than 512 elements. Before an upgrade of the converter, examine issue #197 again.
