# 3. Answer searches of every search index through one endpoint

Status: accepted

## Context

`corpustext` writes crawled pages into a search index, and the SearXNG engine
`searxng-crawled-text-search` read them back by querying Elasticsearch or
Manticore directly. The engine therefore restated what `corpustext` owns: the
index names, the schema version, the fields, and the ranking. A schema version
change in `corpustext` left the engine reading old indexes until an operator
changed its settings too.

A search index that runs inside `corpustext` has no endpoint of its own for a
client to query. Either every search index answers through `corpustext`, or
none does.

## Decision

`corpustext` serves `GET /search` and answers it from whichever search index it
writes to. Each search index adapter builds the query for its engine. The
ranking — field weights, every word must match, snippet length — lives once in
`corpustext` and every adapter applies it.

The SearXNG engine queries `corpustext` only and knows no search index.

## Consequences

- Index names, schema, and ranking each live in one place.
- Changing the search index changes the configuration of `corpustext` only.
- Local search stops while `corpustext` stops, also for Elasticsearch and
  Manticore.
- A search feature the endpoint does not offer needs a change to the endpoint
  and to each adapter.
- The query code for Elasticsearch and Manticore moved from Python to Go.
