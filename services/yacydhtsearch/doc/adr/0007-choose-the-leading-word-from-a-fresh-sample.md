# 7. Choose the leading word from a fresh sample

Date: 2026-09-23

## Status

Accepted

## Context

Today a query asks every word in all 16 partitions. Only the rarest word, the leading
word, needs all of them. A document joins only in the partition of its URL hash, so the
other words are needed only in partitions where the leading word has documents.

The cross-check round asks again, with `urls`, where discovery was incomplete. Real YaCy
peers ignore `urls`, so peer judgements skip them. The round mostly works around the
1000-document abstract cap of `yacynode`.

## Decision

1. Ask every word in one random partition. The rarest word with a complete answer there
   leads. If no word has one, ask every word in all partitions without `urls`.
2. Ask the leading word and the compound words that contain it in all partitions. Their
   documents are the candidates.
3. As each partition answers, ask the other words and compound words there with the
   candidates in `urls`. Skip partitions without candidates.
4. Ask without `urls` when a partition has more candidates than
   `YACYDHTSEARCH_DOCUMENTS_TO_MATCH_CEILING`.
5. Join all answers locally. An answer that ignored `urls` is used as it is.
6. Ask URL metadata only for joined documents that no answer carried metadata for.

Discovery becomes one round. The cross-check round and peer judgements are removed. Counts
that peers claim may now choose which peers are asked. Every count used is fresh, from a
peer that today's plan asks anyway. Nothing is kept between queries.

## Consequences

* Calls stay at or below today. Queries save calls only when the rarest word has fewer
  than about 50 documents; the judged queries of 2026-09-21 save 0.2 % to 3.5 %.
* The sample adds one step before the leading word.
* A partition where every peer failed stays unanswered, because no cross-check retries it.
* Ranking rarity for the other words comes from fewer partitions.
* A stored hint may later replace the sample. It cannot raise the saving.
