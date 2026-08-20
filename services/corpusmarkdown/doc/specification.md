# corpusmarkdown — Technical Specification

## Context

`corpusmarkdown` is a separate, optional, disposable Go service that keeps the latest
markdown representation of an operator's own crawled pages for exact-URL recall.
`yacycrawler` can optionally publish a `markdown` representation of the pages it crawls;
this service consumes that one representation and writes it to a URL-addressed object store.

## Non-Goals

* Serving recall queries or exposing any read API.
* Indexing, ranking, or searching the stored markdown.
* Crawling, fetching, or converting pages — that is `yacycrawler`'s job.
* Storing markdown anywhere other than the operator's own object store.

## Functional Requirements

* The service SHALL consume only the crawler's `markdown` representation stream.
* For each page on that stream, the service SHALL store its markdown under an object name
  derived solely from the page's canonical URL.
* Storing a page whose canonical URL is already stored SHALL overwrite it, so the store
  holds one current copy per URL.
* While the object store is unavailable, the service SHALL drop no page's markdown, resuming
  once the object store returns.
* On an undecodable message the service SHALL halt intake and leave the message pending for
  an operator, rather than discard it.

## Non-Functional Requirements

* The service SHALL keep memory bounded independently of corpus size, capping how many pages
  it stores concurrently.
* The service SHALL keep the markdown compressed on disk, without a change to the bytes
  a reader gets.
* The service SHALL persist no state of its own: the markdown of record lives in the object
  store and any pending backlog lives with the broker.
* The service SHALL be independently disposable: operators MAY stop it and later re-enable
  the crawler's `markdown` representation without depending on this service's prior state.
* The service SHALL support many concurrent instances over the crawler's published pages,
  with each page stored by exactly one instance.
* The service SHALL expose its storage behavior as machine-readable metrics and a liveness
  signal, so operators can observe pages received, stored, and failed, without altering how
  pages are stored.

## Known Limitations

* A URL that is never recrawled is never refreshed and a removed URL is never deleted, so the
  store can hold stale markdown; freshness and deletion scheduling are out of scope.
* While the object store is down, unstored markdown waits on the broker. If the outage lasts
  longer than the broker keeps the message, the broker drops it and the page's markdown is lost
  until the next recrawl.
* If the crawler's canonicalization changes, a page's canonical URL changes with it; with no
  migration here, its old and new objects both persist until an operator intervenes.
