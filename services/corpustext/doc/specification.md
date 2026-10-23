# corpustext — Technical Specification

## Context

`corpustext` is an optional Go service that makes an operator's own crawled corpus
searchable as full text. `pagescrape` reads each page once and offers it; this service
derives the text of the offered page, indexes it into a full-text search index, and
answers searches of that index.

## Non-Goals

* Fetching a page: the scrape service reads the page and offers it.
* Running a search page: a client such as SearXNG shows the results.
* Running or provisioning an Elasticsearch or Manticore server.
* Deciding which pages to index: the crawler decides which pages it reaches.
* Judging a page again: an offered page is already admitted, so this service applies no
  scope, robots, or indexing rule of its own.
* Storing text anywhere other than the configured search index.

## Functional Requirements

* The service SHALL consume only the offered pages of the scrape service.
* For each offered page, the service SHALL derive the text of the document it holds.
* The service SHALL produce a search-index document carrying the offered page's canonical
  URL, text, auxiliary text, title, and language.
* The auxiliary text of a page SHALL be the text that the page has beside its article,
  such as notes.
* An offered page from which no text derives SHALL leave the index unchanged.
* The service SHALL send back a receipt for every page it disposes of, saying whether it
  kept the page or rejected it.
* On an undecodable message the service SHALL halt intake and leave the message pending for
  an operator, rather than discard it.
* Re-indexing a page SHALL overwrite the document of its canonical URL, not add a duplicate.
* Each configured language SHALL get an index with its text analysis; the service SHALL
  write a page into the index of its language, or into one language-neutral index.
* Index names SHALL carry a schema version, which SHALL change only with a
  release of the service.
* The service SHALL search every index of its version through one name per version.
* At startup the service SHALL give each index of its version every field it writes,
  creating the index when it is missing.
* Starting the service SHALL keep every document an index holds.
* A schema change that the documents of an index cannot take SHALL get a new version.
* The service SHALL keep the search index in a local directory unless operators name a
  supported search index server.
* The service SHALL drop no page's content while the search index is unavailable.
* The service SHALL answer a search with the pages whose title, content and auxiliary text
  hold every word of the query, ranked title over content over auxiliary text in every index.
* Each match SHALL carry its URL, title, and a content snippet of at most 300 characters.
* A search SHALL get only pages of the language it names, and the page of at most 50 matches
  it asks for.
* The service SHALL answer searches while the page offers are unavailable, and SHALL try
  again to open them.

## Non-Functional Requirements

* The service SHALL keep memory bounded independently of corpus size, capping how many
  pages it takes in and indexes concurrently.
* The service SHALL be independently disposable: operators MAY stop it and later start it
  again without depending on this service's prior state. Local search stops while the
  service stops.
* With a search index server, the service SHALL persist no state of its own, the server
  holding the index of record and the broker any backlog, and SHALL support many concurrent
  instances over the offered pages, with each page indexed by exactly one instance.
* The service SHALL expose machine-readable metrics and a liveness signal covering pages
  received, indexed, and failed, index latency, searches answered and failed, and search
  latency.

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
* A local index directory serves one instance on one host only.
