# 19. Publish every read page of any media type

Date: 2026-10-23

## Status

Accepted

## Context

The crawler fetched every link its profile admitted, whatever the media type. It then read
the page as HTML. A page that was not HTML ended as `unsupported-media-type` and was not
published, although the fetch succeeded and the crawler knew the page exists.

The specification says that a consumer, not the crawler, decides what a published page is
used for. ADR 12 was superseded for the same reason: each consumer derives the content it
wants. Dropping a page because the crawler cannot find links in it decides for every
consumer.

Other crawlers keep such pages. YaCy indexes a linked document that no parser reads as a
URL with no content (`directDocByURL`, on by default). Heritrix and Common Crawl keep every
response with its media type, and their consumers filter on it.

## Decision

The crawler publishes every page it read, of any media type. A crawled page carries the
page's `ContentType` beside its `PageURL`, as the origin sent it.

A page that is not HTML gives no links. Its indexing refusal comes from its `X-Robots-Tag`
headers, so a document marked `noindex` goes to `crawl.page.indexing-refused`.

The `unsupported-media-type` visit result is removed, because no visit ends with it.

## Consequences

A consumer that reads only HTML now gets pages it cannot read. Today `crawlscrapebridge`
relays them, `pagescrape` fetches them again, and the corpora reject them. That costs one
more fetch for each such page and consumer, up to `YACYCRAWLER_MAX_BODY_BYTES`. A consumer
can skip a page by its `ContentType`, and an extractor for a new format makes those pages
useful with no change to the crawler.

Messages published before this change have no `ContentType`. A consumer reads the empty
value as unknown.

Crawls counted before this change can keep visits under `unsupported-media-type`. The API
still returns that count. The crawl console no longer shows it.
