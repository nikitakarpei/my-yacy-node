# searxng-crawled-text-search — Technical Specification

## Context

`corpustext` indexes crawled pages into an operator's own full-text search index, each
document following the shape defined by `searchdocument`. SearXNG ships a built-in `elasticsearch`
engine, but it renders hits as a generic key/value table with no `url` field — a shape
`searxng-result-router` cannot rewrite, since it only rewrites the `url` field of a standard
result — and it speaks only to Elasticsearch. `searxng-crawled-text-search` is a SearXNG engine
module that queries the search of `corpustext`, whichever search index backs it, and returns
each match as a standard result carrying its title, URL, and matched text, so it composes with
`searxng-result-router` and closes the loop from crawl to index to search to a tracked,
re-crawlable click.

## Non-Goals

* Indexing pages into the search index — that is `corpustext`'s job.
* Rewriting or routing result links — that is `searxng-result-router`'s job.
* Running or provisioning the search index itself.
* Querying a search index directly: `corpustext` does that.
* Reranking, deduplicating, or filtering what `corpustext` returns.

## Functional Requirements

* For each match `corpustext` returns, the engine SHALL return a standard SearXNG result
  carrying the match's title, canonical URL, and snippet, reading the fields `searchdocument`
  defines.
* A result's URL SHALL be the document's canonical URL, unchanged, so a plugin such as
  `searxng-result-router` can still rewrite it.
* The engine SHALL let operators configure the base URL of the `corpustext` search.
* When SearXNG gives a search language, the engine SHALL pass it to `corpustext`.
* The engine SHALL return an empty result set, not an error, when the search terms match no
  document.
* The engine SHALL fetch the page of results SearXNG asks for, consistent with SearXNG's own
  pagination.

## Non-Functional Requirements

* The engine SHALL persist no state of its own between requests.
* The engine SHALL add no observable delay beyond the `corpustext` search itself.
* While `corpustext` or its search index is unavailable, the engine SHALL return no results
  rather than fail the person's entire search.

## Known Limitations

* Relevance ordering is whatever `corpustext` returns; the engine reranks nothing of its own.
* The engine depends on the search contract `searchdocument` defines; a change there is a change
  to this engine's contract, not something its configuration can absorb.
