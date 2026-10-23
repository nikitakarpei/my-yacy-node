# 5. Keep the search index in Bleve unless an operator names an engine

Status: accepted

## Context

`corpustext` needed an Elasticsearch or Manticore server before it could index a single
page, so an operator with a small corpus had to run a search server only to try it.
Since ADR 3, `corpustext` answers searches itself, so an index inside the service needs
no endpoint of its own.

Bleve is a full-text search library in pure Go, under the Apache 2.0 licence, with
stemming for English, German, French, and Russian. SQLite FTS5 stems English only.

## Decision

Add `github.com/blevesearch/bleve/v2`. `SEARCH_INDEX_ENGINE` defaults to `bleve`, which
keeps the index in `BLEVE_DIRECTORY`. Elasticsearch and Manticore stay available.

## Consequences

- The service runs with no search server.
- With Bleve, one instance owns the directory, and the service keeps state of its own.
- The service gains the Bleve modules as dependencies.
- Ranking differs between engines.
