# 12. Publish every representation a page format derives

Date: 2026-08-06

## Status

Accepted

## Context

The publisher derived every enabled representation from one page and failed the page when any
target format was unreachable. Startup validated reachability from a single constant source
format, so the check proved nothing about a second content type. Together these made a page whose
format reaches only some representations abort the whole crawl order at run time, on a
configuration that started cleanly.

An absent derivation is a decision, not a limitation. Text extracted from a document does not
reach markdown today, yet one derivation edge would close that gap. The derivation catalog is the
one place that records which format feeds which representation. A rule that demands every format
reach every representation removes that choice: each gap must be closed with a derivation that
restates plain text as markup, or the format cannot be admitted at all.

## Decision

Publication is per representation. The publisher sends every representation the page format
derives, counts the ones it cannot as `RepresentationUnderivable`, and skips them. A missing
derivation never ends a crawl order. A page that derives no representation at all fails
publication with an error: the startup rules leave that state reachable only by an extractor that
emits a format it does not declare, and an error keeps the page unacknowledged instead of dropped.

The composition root collects the format each admitted extractor emits and validates two rules at
startup: every enabled representation is derivable from at least one emitted format, and every
emitted format derives at least one enabled representation. A media type in
`YACYCRAWLER_CONTENT_TYPES` that no registered extractor or expander reads fails startup.

Extractors and container expanders are registered in `mediaExtractorCatalog` and
`containerExpanderCatalog`. Each extractor declares the media types it reads and the format it
emits.

## Consequences

Streams no longer carry the same pages: a representation receives only pages whose format derives
it, and a consumer must not assume two streams hold the same URLs. Coverage per representation is
observable through `yacycrawler_representations_underivable_total`. One new derivation edge
closes a gap for every future page of that format; no publisher change is necessary.

A dead configuration fails at startup rather than at run time. A new content type is a plugin
package and one catalog line; whether it reaches every representation is a property of the
derivations it registers, not a condition of admitting it.
