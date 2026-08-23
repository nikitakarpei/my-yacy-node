# 5. Extract one document per fetch

Date: 2026-08-21

## Status

Accepted

## Context

The crawler no longer derives content. It publishes the URL of each page it reached, and each
interested service fetches that URL again and derives the format it wants.

Container expansion does not survive that change. A reached page names one URL, but a container
holds many documents under synthetic member URLs of the form `containerURL!/path`. No client can
fetch a member URL, so each service must expand the container again to find the same members.
The corpus then holds entries under addresses that no reader can open.

The mechanism also gives very little. Only HTML has a registered extractor, so a container must
hold HTML for the expansion to produce anything, and archives of HTML pages are rare on the web.
Against that, expansion costs a recursive router, a nesting bound, a document bound, a member
size bound, member name collision rules, and zip, tar, and gzip readers.

## Decision

One fetch yields at most one document. `contentextraction` keeps its media-type router and
returns the document that the registered extractor reads, or `ErrUnsupportedMediaType`. The
`ContainerExpander` port, the `ContainerMember` type, the `archive` expander, and the nesting and
document bounds are removed. `pagescrape.Scrape` returns one page and a flag that tells whether
the fetch yielded one.

Archive media types are no longer registered, so `YACYCRAWLER_CONTENT_TYPES` refuses them.

## Consequences

A fetched archive is disposed as `unsupported-media-type`. Links that a member document held no
longer reach the frontier. The corpus holds only URLs that a client can fetch.

The `nesting-too-deep` and `document-budget-exhausted` disposal reasons are removed, because
nothing can cause them. A decompression bomb is no longer a risk in extraction: the body size
limit is the only bound that extraction needs.

Support for a container format is possible again later, but it must first answer how a member is
addressed so that a reader can fetch it.
