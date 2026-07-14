# yacycrawler — Technical Specification

## Context

`yacycrawler` is a standalone, optional, disposable crawling service. It accepts crawl
orders, fetches the web pages they reach, and publishes representations of them through
the message-broker API defined in `yacycrawlcontract`. A YaCy node is the typical order
source and consumer, but the service depends on no consumer's internals.

Several instances can share one order stream, each order running on one instance. The
service is meant for a more capable host than an always-on node.

## Non-Goals

* Producing the crawl orders it consumes, or consuming its own published representations.
* Participating in the YaCy DHT peer protocol.
* Ranking, indexing, or judging what it fetches.
* Authorizing broker subjects beyond the broker deployment's own trust boundary.
* Defeating anti-bot walls; a wall is a refusal signal to honor, not an obstacle to evade.
* Guaranteeing delivery beyond the broker's own semantics.

## Functional Requirements

* The service SHALL idle until a crawl order arrives, then process it.
* The service SHALL crawl only what an order's profile admits, from its seeds and
  discovered links.
* Every crawl run SHALL terminate within a run-wide page budget, never by elapsed time.
* Before fetching a page, the service SHALL ask the recrawl decision whether it is due,
  and skip the fetch if not.
* Every outbound fetch SHALL egress through the operator's configured proxy.
* The service SHALL honor a target's explicit refusal, ceasing or deferring the fetch
  rather than pressing against it.
* The service SHALL publish representations of a page, each on its own stream: the `rwi`
  representation, carrying page references and never a body, and content representations,
  each additive and operator-selected.
* The service SHALL publish each page to every enabled representation that accepts its
  content format, advancing those together: if any cannot take it yet, the others wait.
* Every fetched page SHALL reach one terminal outcome: published to every representation
  accepting its content format, or disposed per operator policy.
* A publication SHALL fail only on a hard, non-retryable broker error; transient
  backpressure waits for as long as the run holds its order.
* A publication failure SHALL NOT be terminal; the page stays unpublished.
* The service SHALL acknowledge an order only after every page it produced reached a
  terminal outcome.

## Non-Functional Requirements

* The service SHALL process each order idempotently per its identity under at-least-once
  delivery.
* The service SHOULD assert continued ownership of an in-progress order to reduce
  redundant concurrent runs; correctness never depends on it holding.
* Each published page SHALL be addressed by its canonical URL, so a re-run of that URL
  replaces its prior publication downstream rather than duplicating it.
* The service SHALL cap every resource an order can inflate — its frontier, buffers, and
  fetched-body sizes — keeping memory bounded regardless of run size.
* The service SHALL bound every outbound fetch with an explicit deadline.
* The core SHALL keep no state of its own; anything remembered between runs lives behind an
  interface it consults. A run survives a restart only if the order source resends the order.
* The message broker SHALL be replaceable behind a narrow interface assuming at-least-once
  delivery with acknowledgment and redelivery, with no change to crawl logic.
* The page-fetch mechanism SHALL be replaceable behind a narrow interface, with no
  change to crawl logic.
* Deriving a representation SHALL sit behind a narrow interface declaring the content
  formats it accepts, so adding a representation changes no crawl logic.
* A representation SHALL stay whole in the domain; framing it into bounded wire messages
  is the publish edge's concern.
* The recrawl decision SHALL sit behind a narrow interface; its default admits every page.
* Operational behavior SHALL be observable through machine-readable metrics.

## Known Limitations

* Target-safety (internal-host targeting, DNS rebinding), per-host crawl politeness,
  robots obedience, and page rendering all depend entirely on the configured proxy's
  policy; the service adds none of them.
* A proxy that rewrites fetch responses must preserve the refusal and wait signals the
  service honors, or the service cannot act on them.
* Content representations carry renderable web content that anyone with broker publish
  rights can inject; restrict publish rights on their subjects to the crawler.
* A consumer's outage-survival for a content representation holds only within the
  retention window the operator sizes.
