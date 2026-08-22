# corpusmarkdown — Technical Specification

## Context

`corpusmarkdown` is a separate, optional, disposable Go service that keeps the latest
markdown of an operator's own crawled pages for exact-URL recall. `yacycrawler` publishes
the URL of every page it reaches; this service fetches each of those pages through the
egress proxy, derives markdown from it, writes the markdown to a URL-addressed object
store, and serves that markdown back to callers that ask for one URL.

## Non-Goals

* Crawling on demand, or waiting for a page the corpus does not hold yet.
* Indexing, ranking, or searching the stored markdown.
* Deciding which pages to store: the crawler decides which pages it reaches.
* Judging a page again: a scrape request is already admitted, so this service applies no
  scope, robots, or indexing rule of its own.
* Storing markdown anywhere other than the operator's own object store.

## Functional Requirements

* The service SHALL consume only the crawler's scrape-request stream.
* For each scrape request, the service SHALL fetch the page through its configured proxy and
  derive markdown from the document it holds.
* The service SHALL store the derived markdown under an object name derived solely from the
  fetched page's canonical URL.
* Storing a page whose canonical URL is already stored SHALL overwrite it, so the store
  holds one current copy per URL.
* A scrape request that the service cannot read, or from which no markdown derives, SHALL
  leave the store unchanged and SHALL NOT be fetched again for that message.
* While the fetch or the object store is unavailable, the service SHALL drop no page,
  resuming once it returns.
* The service SHALL serve the markdown it holds for a requested URL over gRPC.
* For a URL the service holds no markdown for, it SHALL answer that the corpus holds none,
  and SHALL neither fetch the page nor order a crawl.
* On an undecodable message the service SHALL halt intake and leave the message pending for
  an operator, rather than discard it.

## Non-Functional Requirements

* The service SHALL keep memory bounded independently of corpus size, capping how many pages
  it fetches and stores concurrently.
* The service SHALL bound each fetch by a maximum body size and a deadline.
* The service SHALL keep the markdown compressed on disk, without a change to the bytes
  a reader gets.
* The service SHALL persist no state of its own: the markdown of record lives in the object
  store and any pending backlog lives with the broker.
* The service SHALL be independently disposable: operators MAY stop it and later start it
  again without depending on this service's prior state.
* The service SHALL support many concurrent instances over the crawler's scrape requests,
  with each page stored by exactly one instance.
* The service SHALL expose its behavior as machine-readable metrics and a liveness signal,
  so operators can observe pages received, stored, and failed, without altering how pages
  are stored.

## Known Limitations

* The service fetches each page again after the crawler did. The origin therefore serves the
  page more than once, and the markdown records what this service fetched, which can differ
  from what the crawler read.
* A URL that is never recrawled is never refreshed and a removed URL is never deleted, so the
  store can hold stale markdown; freshness and deletion scheduling are out of scope.
* While the object store is down, unstored markdown waits on the broker. If the outage lasts
  longer than the broker keeps the message, the broker drops it and the page's markdown is lost
  until the next recrawl.
* If canonicalization changes, a page's canonical URL changes with it; with no migration here,
  its old and new objects both persist until an operator intervenes.
