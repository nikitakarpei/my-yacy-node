# corpustext — Technical Specification

## Context

`corpustext` is an optional Go service that makes an operator's own crawled corpus
searchable as full text. `yacycrawler` publishes the URL of every page it reaches; this
service fetches each of those pages through the egress proxy, derives its text, and
indexes it into an operator-run full-text search index.

## Non-Goals

* Serving search queries.
* Running or provisioning the search index server itself.
* Deciding which pages to index: the crawler decides which pages it reaches.
* Judging a page again: a scrape request is already admitted, so this service applies no
  scope, robots, or indexing rule of its own.
* Storing text anywhere other than the operator's own search index.

## Functional Requirements

* For each scrape request, the service SHALL fetch the page through its configured proxy and
  derive the text of the document it holds.
* The service SHALL produce a search-index document carrying the fetched page's canonical
  URL, text, title, and language.
* A scrape request that the service cannot read, or from which no text derives, SHALL leave
  the index unchanged and SHALL NOT be fetched again for that message.
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
  pages it fetches and indexes concurrently.
* The service SHALL bound each fetch by a maximum body size and a deadline.
* The service SHALL persist no state of its own: the index of record lives in the search
  index and any pending backlog lives with the broker.
* The service SHALL be independently disposable: operators MAY stop it and later start it
  again without depending on this service's prior state.
* The service SHALL support many concurrent instances over the crawler's scrape requests,
  with each page indexed by exactly one instance.
* The service SHALL expose machine-readable metrics and a liveness signal covering pages
  received, indexed, and failed, and index latency.

## Known Limitations

* The service fetches each page again after the crawler did. The origin therefore serves the
  page more than once, and the index records what this service fetched, which can differ
  from what the crawler read.
* A URL that is never recrawled is never refreshed, and a removed URL is never deleted, so
  the index can hold stale documents.
* Content held longer than the broker's retention while the search index is down stays
  unindexed until a recrawl.
* If canonicalization changes, a page gets a new canonical URL, and its old
  and new documents both stay in the index until an operator intervenes.
* A change of the configured languages or a new version starts new, empty indexes;
  earlier documents stay in the old indexes until a recrawl or reindex.
* Deleting the indexes of an old version is the operator's task.
