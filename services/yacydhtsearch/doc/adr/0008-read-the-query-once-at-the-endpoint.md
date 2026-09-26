# 8. Read the query once, at the endpoint

Date: 2026-09-26

## Status

Proposed

## Context

The ranking cache spells the key of a ranking from the query: its words, its exclusions, and
its language. A compound word comes from a run of adjacent words that are not stopwords. A
stopword ends a run. The key does not show where a run ends.

Thus `state art` and `state of the art` have one key. Only the first asks for `stateart`, but
each query can get the ranking of the other.

Without a client language, the service guesses the language from the stopwords of the query.
Thus the words that stay after the stopwords go can give a different query when the service
reads them again. A key made from them is not a safe name for the query.

## Decision

1. A search goes through three stages in one process. Each stage calls the next through one
   interface. The interface takes a query and gives a ranking and the outcome of the search.
2. The endpoint reads the text and the language that the client sent. It removes the
   stopwords and keeps the end of each run. The result is the canonical query. Only the
   endpoint knows stopwords and client text.
3. The canonical query holds the runs in order, the exclusions, and the language that the
   client sent. The guessed language does not go into it. A second read of its spelling
   gives the same query.
4. The endpoint cuts the page from the ranking that the next stage gives, as ADR 2 says.
5. The ranking cache wraps the network search. It uses the spelling of the canonical query
   as the key and does not read it. It keeps a ranking only when the outcome shows that
   peers were asked. It keeps rankings in memory or in NATS, as ADR 3 and ADR 4 say.
6. The network search reads the canonical query without stopwords. It makes the compound
   words from the runs and asks the peers.

## Consequences

* Queries that have the same runs share one ranking, also when their stopwords differ.
* Queries that ask for different compound words do not share a ranking.
* The keys change. Rankings in NATS under the old keys are not found and expire after
  `YACYDHTSEARCH_RANKING_LIFETIME`.
* The ranking cache and the network search do not know stopwords. A change to the stopword
  lists changes only the endpoint.
* ADR 2, ADR 3, and ADR 4 stay valid.
