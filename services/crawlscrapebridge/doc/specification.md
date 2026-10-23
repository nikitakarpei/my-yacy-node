# crawlscrapebridge — Technical Specification

## Context

`yacycrawler` publishes each page it crawls. `pagescrape` reads a page only when it gets a
scrape request for that page. `crawlscrapebridge` is a disposable Go service that asks
`pagescrape` to scrape each page the crawler found indexable, so every corpus gets the
pages the crawler visits.

## Non-Goals

* Fetching, parsing, or storing a page.
* Deciding which pages the crawler visits or which pages are indexable.
* Relaying a page the crawler did not find indexable.

## Functional Requirements

* The service SHALL publish one scrape request for each indexable page the crawler
  publishes.
* The service SHALL acknowledge a crawled page only after its scrape request is published.
* The service SHALL return a crawled page for a later try when its scrape request is not
  published.
* The service SHALL stop relaying, and leave the crawled page pending, when it cannot read
  a crawled page.
* The service SHALL relay the crawled pages published while it was stopped.
* The service SHALL start while the crawled pages are unavailable, and SHALL try again to
  open them.

## Non-Functional Requirements

* The service SHALL persist no state of its own; the broker keeps the position of its
  durable consumer.
* The service SHALL relay at most the configured number of crawled pages at once.
* The service SHALL expose machine-readable metrics, including scrape requests published
  and not published.

## Known Limitations

* A crawled page held longer than the crawler's stream keeps it, while the service is
  stopped, is not relayed.
* A page the crawler publishes twice gets two scrape requests.
