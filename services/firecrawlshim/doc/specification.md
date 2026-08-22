# firecrawlshim — Technical Specification

## Context

`firecrawlshim` is an edge adapter that exposes the Firecrawl v1 scrape API over HTTP. For
each call it orders a crawl of the URL and waits until the operator's own corpus holds the
markdown of the page. It lets a client written against Firecrawl read that corpus without
a change to the client.

## Non-Goals

* Fetching, rendering, parsing, or extracting pages — the crawler and the corpus do that.
* Holding a corpus or any state of its own.
* Reproducing Firecrawl features beyond single-URL scrape.

## Functional Requirements

* The service SHALL accept `POST /v1/scrape` naming one URL.
* The service SHALL place one crawl order for the URL of each accepted scrape.
* The service SHALL read what crawling last did with the URL, and ask the corpus for the
  markdown of the URL that crawling resolved it to.
* The service SHALL return the markdown in the Firecrawl scrape response shape, carrying
  the page's canonical URL as its source URL.
* The service SHALL stop the wait as soon as crawling disposes of the page after the
  order, and report the reason to the caller.
* The service SHALL report the markdown unavailable when the corpus does not hold it
  within the recall limit.
* The service SHALL reject a request that names no URL or that it cannot decode.
* The service SHALL report a collaborator that it cannot reach as a gateway error.

## Non-Functional Requirements

* The service SHALL persist no state of its own between requests.
* The crawler and corpus targets SHALL be configurable, and the wait SHALL be bounded.
* The service SHALL bound the number of scrapes that wait at the same time, and refuse a
  scrape beyond that bound.

## Known Limitations

* Only markdown is served; a request that names another Firecrawl format gets markdown.
* The response carries no title and no language, because the markdown corpus holds
  neither.
* A page the corpus gains after the recall limit is served only to a later scrape of the
  same URL.
