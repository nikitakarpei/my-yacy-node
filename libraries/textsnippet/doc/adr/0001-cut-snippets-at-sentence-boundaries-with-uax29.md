# 1. Cut snippets at sentence boundaries with clipperhouse/uax29

Date: 2026-10-24

## Status

Accepted

## Context

yacydhtsearch and corpustext both show a snippet of a page for a query. Each kept its own
rule, so the two showed different passages for the same page. A passage that starts or
ends inside a sentence starts or ends mid-thought, and the pages come in many languages.

## Decision

The library chooses the snippet for both services. It finds sentence boundaries with
`github.com/clipperhouse/uax29/v2/sentences`, pinned in `go.mod`, which applies the
Unicode UAX 29 sentence rules from tables alone, holds no model, carries the MIT license
and requires no other module. A service names how its text splits into words, so each
service matches query words as its own search does.

## Consequences

Both services show the same kind of passage. An abbreviation such as `z.B.` or `Dr.` ends
a sentence in the rules, so a snippet can end a few characters early after one. A change
to the rule changes both services at their next build.
