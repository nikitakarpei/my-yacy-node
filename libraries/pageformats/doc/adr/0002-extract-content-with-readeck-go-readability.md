# 2. Extract page content with readeck/go-readability

Date: 2026-07-06

## Status

Accepted

## Context

A page's postings should reflect its main content, not its boilerplate. We need a
Readability.js-equivalent extractor that finds the main article of a page.

## Decision

We use `codeberg.org/readeck/go-readability/v2` (pinned in `go.mod`) to find the main article
of a page. The `readablehtml` derivation calls `ParseAndMutate` on its own copy of the parsed
tree, thus the tree that the document holds does not change. The article tree is the
readable-html body.

## Consequences

Readable text and markdown come from the article, which keeps boilerplate out of postings. When
readability finds no article text, readable text comes from the full text of the page, and
markdown comes from the whole document. The extractor is confined to the `readablehtml`
derivation.

Readability renames elements and adds elements without an atom. The derivation sets the atom of
each such html element from its tag name, so the article tree reads like a parsed tree.
