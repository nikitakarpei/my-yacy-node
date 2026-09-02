# corpustext — Technical Specification

## Context

`corpustext` is an optional Go service that makes an operator's own crawled corpus
searchable as full text. `pagescrape` reads each page once and offers it; this service
derives the text of the offered page and indexes it into an operator-run full-text search
index.

## Non-Goals

* Fetching a page: the scrape service reads the page and offers it.
* Serving search queries.
* Running or provisioning the search index server itself.
* Deciding which pages to index: the crawler decides which pages it reaches.
* Judging a page again: an offered page is already admitted, so this service applies no
  scope, robots, or indexing rule of its own.
* Storing text anywhere other than the operator's own search index.

## Functional Requirements

* The service SHALL consume only the offered pages of the scrape service.
* For each offered page, the service SHALL derive the text of the document it holds.
* The service SHALL produce a search-index document carrying the offered page's canonical
  URL, text, title, and language.
* An offered page from which no text derives SHALL leave the index unchanged.
* The service SHALL send back a receipt for every page it disposes of, saying whether it
  kept the page or rejected it.
* On an undecodable message the service SHALL halt intake and leave the message pending for
  an operator, rather than discard it.
* Re-indexing a page whose canonical URL is already indexed SHALL overwrite that document
  rather than add a duplicate.
* Each configured language SHALL get an index with that language's text analysis.
* The service SHALL write a page into the index of its language, or into one
  language-neutral index.
* Index names SHALL carry a schema version, which SHALL change only with a
  release of the service.
* The service SHALL publish one name per version that a search client reads
  every index of that version through.
* At startup the service SHALL create each missing index and SHALL NOT change or delete
  an existing index.
* The service SHALL let operators choose which supported search index to use.
* The service SHALL drop no page's content while the search index is unavailable.

## Non-Functional Requirements

* The service SHALL keep memory bounded independently of corpus size, capping how many
  pages it takes in and indexes concurrently.
* The service SHALL persist no state of its own: the index of record lives in the search
  index and any pending backlog lives with the broker.
* The service SHALL be independently disposable: operators MAY stop it and later start it
  again without depending on this service's prior state.
* The service SHALL support many concurrent instances over the offered pages, with each
  page indexed by exactly one instance.
* The service SHALL expose machine-readable metrics and a liveness signal covering pages
  received, indexed, and failed, and index latency.

## Known Limitations

* A receipt is not kept. A listener that is away when the service sends one never learns it.
* A URL that is never recrawled is never refreshed, and a removed URL is never deleted, so
  the index can hold stale documents.
* Content held longer than the broker's retention while the search index is down stays
  unindexed until a recrawl.
* If canonicalization changes, a page gets a new canonical URL, and its old
  and new documents both stay in the index until an operator intervenes.
* A change of the configured languages or a new version starts new, empty indexes;
  earlier documents stay in the old indexes until a recrawl or reindex.
* Deleting the indexes of an old version is the operator's task.
