# 8. Read the query once, at the endpoint

Date: 2026-09-26

## Status

Accepted

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

The endpoint is the only stage that sees what the client typed. It drops the stopwords, finds
the compound words, and hands on a canonical query. It also cuts the requested page out of the
ranking, as ADR 2 describes.

A compound word is two or three adjacent words spelled as one, and a stopword between them
prevents it. Finding compounds is part of reading the text, like dropping stopwords, so it
happens in the same place.

The canonical query holds the words, each compound word with the words it is made of, the
exclusions, and the language exactly as the client sent it. Two-word compounds come before
three-word ones. The guessed language only picks the stopword list and goes no further.
No two different queries share the spelling of a canonical query.

The ranking cache wraps the network search. It uses the canonical query's spelling as an
opaque key and never looks inside it. It only keeps a ranking when peers were actually asked,
and it stores rankings in memory or in NATS, as ADR 3 and ADR 4 describe.

The network search reads the canonical query without needing stopwords or the rule that finds
compounds. It still needs each compound and its parts: it asks the peers for the compound, and
it counts a document found under the compound as holding each part. It asks for compounds in
the order given, up to `YACYDHTSEARCH_COMPOUND_WORDS_CEILING`.

## Consequences

Queries that differ only in their stopwords now share a ranking, and queries that ask for
different compound words no longer do.

Reading the text lives in one place. Changing the stopword lists or the compound rule only
touches the endpoint; the cache and the network search do not depend on either.

The keys change on upgrade. Rankings stored in NATS under the old keys are simply never found
again and expire after `YACYDHTSEARCH_RANKING_LIFETIME`.

ADR 2, ADR 3 and ADR 4 remain in force.
