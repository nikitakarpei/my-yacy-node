# 5. Keep the search index in Bleve unless an operator names an engine

Status: accepted

## Context

`corpustext` needed an Elasticsearch or Manticore server before it could index a
single page. An operator with a small corpus had to run, upgrade, and back up a
search server only to try the service. Every service should run as its binary
alone and handle a low workload.

Since ADR 3, `corpustext` answers searches itself, so a search index inside the
service needs no endpoint of its own.

Bleve is a full-text search library in pure Go, under the Apache 2.0 licence. It
has analyzers with stemming and stopwords for English, German, French, and
Russian, field boosts, highlighted fragments, and an alias that searches many
indexes at once. Its default storage returns from a write only when the write
is on disk.

SQLite FTS5 was the other choice. Its only stemmer is for English.

## Decision

Add `github.com/blevesearch/bleve/v2` as a dependency. `SEARCH_INDEX_ENGINE`
defaults to `bleve`, and the service keeps one Bleve index per configured
language, plus one for all other pages, under `BLEVE_DIRECTORY`. Elasticsearch
and Manticore stay available when an operator names them.

## Consequences

- The service runs with no search server.
- A Bleve directory belongs to one process: with Bleve, one instance runs, on
  one host, and the service keeps state of its own.
- The dependency tree of the service grows by the Bleve modules.
- Ranking differs between Bleve and the other engines, as it does between
  Elasticsearch and Manticore.
