# corpustext — Technical Specification

## Context

`corpustext` is an optional Go service that makes an operator's own crawled corpus
searchable as full text. It consumes the `text` representation `yacycrawler` publishes for
the pages it crawls and indexes it into an operator-run full-text search index.

## Non-Goals

* Serving search queries.
* Running or provisioning the search index server itself.
* Crawling, fetching, or parsing pages — that is `yacycrawler`'s job.
* Storing text anywhere other than the operator's own search index.

## Functional Requirements

* The service SHALL consume only the crawler's `text` representation stream.
* For each page on that stream, the service SHALL produce a search-index document
  preserving its canonical URL, text, and metadata.
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
  documents it indexes concurrently.
* The service SHALL persist no state of its own: the index of record lives in the search
  index and any pending backlog lives with the broker.
* The service SHALL be independently disposable: operators MAY stop it and later re-enable
  the crawler's `text` representation without depending on this service's prior state.
* The service SHALL support many concurrent instances over the crawler's published pages,
  with each page indexed by exactly one instance.
* The service SHALL expose machine-readable metrics and a liveness signal covering pages
  received, indexed, disposed, and failed, and index latency.

## Known Limitations

* A URL that is never recrawled is never refreshed, and a removed URL is never deleted, so
  the index can hold stale documents.
* Content held longer than the broker's retention while the search index is down stays
  unindexed until a recrawl.
* If the crawler's canonicalization changes, a page gets a new canonical URL, and its old
  and new documents both stay in the index until an operator intervenes.
* A change of the configured languages or a new version starts new, empty indexes;
  earlier documents stay in the old indexes until a recrawl or reindex.
* Deleting the indexes of an old version is the operator's task.
