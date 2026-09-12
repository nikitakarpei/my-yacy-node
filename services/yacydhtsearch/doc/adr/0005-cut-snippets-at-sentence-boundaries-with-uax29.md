# 5. Cut snippets at sentence boundaries with clipperhouse/uax29

Date: 2026-09-12

## Status

Accepted

## Context

The snippet of a result is cut from the text of its page. A cut inside a sentence shows a
fragment that starts and ends mid-thought. The pages come in many languages, so the boundary
rule cannot depend on a model per language.

## Decision

The service finds sentence boundaries with `github.com/clipperhouse/uax29/v2/sentences`,
pinned in `go.mod`. It applies the Unicode UAX 29 sentence rules from tables alone, holds no
model, carries the MIT license and requires no other module. The word rule of the service
stays in `yacymodel`; the library is confined to the snippet.

## Consequences

A snippet starts and ends on a sentence boundary the same way in every language. An
abbreviation such as `z.B.` or `Dr.` ends a sentence in the rules, so a snippet can cut a
few characters early after one. A language-aware splitter would need a model per language
and a new decision.
