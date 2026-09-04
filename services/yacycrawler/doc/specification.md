# yacycrawler — Technical Specification

## Context

`yacycrawler` is a standalone, optional, disposable crawling service. It accepts crawl
orders. For each order, it fetches the pages the order admits and publishes each page it
read. A YaCy node is the typical order source and consumer, but the service depends on no
consumer's internals.

Several instances share one order stream and one frontier. Every instance takes any URL
of any order, so a run spreads across the instances that are up. The service is meant for
a more capable host than an always-on node.

## Non-Goals

* Participating in the YaCy DHT peer protocol.
* Ranking, indexing, or judging what it fetches.
* Deriving, carrying, or storing page content for a consumer.
* Authorizing broker subjects beyond the broker deployment's own trust boundary.
* Deciding what a published page is used for; a consumer decides that.
* Defeating anti-bot walls; a wall is a refusal signal to honor, not an obstacle to evade.
* Guaranteeing delivery beyond the broker's own semantics.

## Functional Requirements

* The service SHALL idle until a crawl order arrives, then accept it.
* The service SHALL crawl only what an order's profile admits, from its seeds and
  discovered links.
* A crawl run SHALL end when its profile admits no more URLs, never by elapsed time.
* Before fetching a page, the service SHALL ask the recrawl rule whether the page is
  due, and skip the fetch if not.
* Every outbound fetch SHALL egress through the operator's configured proxy.
* The service SHALL honor a target's explicit refusal, ceasing or deferring the fetch
  rather than pressing against it.
* The service SHALL publish a page that refuses indexing, marked as refusing indexing, so
  that a consumer, not the crawler, decides what to do with it.
* The service SHALL publish the canonical URL of each page it read, and never the page's
  content.
* Every URL a run admits SHALL be visited by exactly one instance, and SHALL reach one
  terminal outcome: published as a crawled page, or disposed, counted against the reason
  for it.
* A publication SHALL fail only on a hard, non-retryable broker error; transient
  backpressure returns the URL for redelivery.
* A publication failure SHALL NOT be terminal; the page stays unpublished.
* The service SHALL acknowledge an order once the order and its seed URLs are durable.

## Non-Functional Requirements

* The service SHALL process each order idempotently per its identity under at-least-once
  delivery.
* Each published page SHALL be addressed by its canonical URL, so a re-run of that URL
  replaces its prior publication downstream rather than duplicating it.
* The service SHALL keep its frontier in the broker, not in memory, so run size never
  inflates an instance; it SHALL cap the buffers and fetched-body sizes it does hold.
* The service SHALL bound every outbound fetch with an explicit deadline.
* The core SHALL keep no state of its own; anything remembered lives behind an interface
  it consults. A run survives the restart of any instance, and of all of them.
* The message broker SHALL be replaceable behind a narrow interface assuming at-least-once
  delivery with acknowledgment and redelivery, with no change to crawl logic.
* The page-fetch mechanism SHALL be replaceable behind a narrow interface, with no
  change to crawl logic.
* The visited pages SHALL sit behind a narrow interface; the default keeps none, so the
  recrawl rule holds every page due.
* Operational behavior SHALL be observable through machine-readable metrics.

## Known Limitations

* Target-safety (internal-host targeting, DNS rebinding), per-host crawl politeness,
  robots obedience, and page rendering all depend entirely on the configured proxy's
  policy; the service adds none of them.
* A proxy that rewrites fetch responses must preserve the refusal and wait signals the
  service honors, or the service cannot act on them.
* A crawled page names a URL anyone with broker publish rights can inject, sending every
  consumer to fetch it; restrict publish rights on the crawled-page stream to the crawler.
* Each consumer fetches the page again for itself, so the origin serves it once per
  interested consumer and each consumer can see a different page than the crawler saw.
