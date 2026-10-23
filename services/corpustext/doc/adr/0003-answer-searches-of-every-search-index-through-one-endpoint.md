# 3. Answer searches of every search index through one endpoint

Status: accepted

## Context

The SearXNG engine `searxng-crawled-text-search` queried Elasticsearch or Manticore
directly, so it repeated what `corpustext` owns: index names, schema version, fields,
and ranking. A search index that runs inside `corpustext` has no endpoint for the
engine to query.

## Decision

`corpustext` serves `GET /search` and answers it from whichever search index it writes
to, with one ranking for every index. The SearXNG engine queries `corpustext` only.

## Consequences

- Index names, schema, and ranking live in one place.
- To change the search index, an operator configures `corpustext` only.
- Local search stops while `corpustext` stops.
- A new search feature needs a change in `corpustext` for each search index.
