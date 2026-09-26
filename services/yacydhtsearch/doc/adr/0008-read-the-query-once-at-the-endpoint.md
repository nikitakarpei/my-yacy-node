# 8. Read the query once, at the endpoint

Date: 2026-09-26

## Status

Proposed

## Context

The ranking cache builds its key from the query's words, exclusions and language. But the
search also asks for compound words, which it builds from runs of adjacent words, and a
stopword breaks a run. The key loses those breaks, so `state art` and `state of the art` share
a key even though only the first asks for `stateart`. Either one can be answered with the
other's ranking.

Reading the stripped words a second time is not safe either. When the client sends no
language, the service guesses one from the stopwords in the query, and once they are gone the
guess, and so the query, can come out differently.

So the cache cannot compute a correct key without knowing how a query is read, and it should
not have to know that.

## Decision

A search passes through three stages in one process, each wrapping the next behind the same
interface: give it a query, get back a ranking and the outcome of the search.

The endpoint is the only stage that sees what the client typed. It drops the stopwords, keeps
the places where they broke a run, and hands on a canonical query. It also cuts the requested
page out of the ranking, as ADR 2 describes.

The canonical query holds the runs in order, the exclusions, and the language exactly as the
client sent it. The guessed language only picks the stopword list and goes no further.
Reading the canonical query again gives back the same query.

The ranking cache wraps the network search. It uses the canonical query's spelling as an
opaque key and never looks inside it. It only keeps a ranking when peers were actually asked,
and it stores rankings in memory or in NATS, as ADR 3 and ADR 4 describe.

The network search reads the canonical query without needing stopwords, builds compound words
from its runs, and asks the peers.

## Consequences

Queries that differ only in their stopwords now share a ranking, and queries that ask for
different compound words no longer do.

Stopwords live in one place. The cache and the network search no longer depend on them, so
changing the stopword lists only touches the endpoint.

The keys change on upgrade. Rankings stored in NATS under the old keys are simply never found
again and expire after `YACYDHTSEARCH_RANKING_LIFETIME`.

ADR 2, ADR 3 and ADR 4 remain in force.
