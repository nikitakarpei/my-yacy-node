# 9. Choose the leading word only from remembered amounts

Date: 2026-10-06

## Status

Accepted. Supersedes step 1 of
[ADR 7](0007-choose-the-leading-word-from-a-fresh-sample.md).

## Context

ADR 7 asks every word in one partition before it asks the leading word. A query
that counts its words this way asks the peers three times in a row: the sample,
the leading word, then the other words.

Each spread measures, from all its answers, the amount of documents of each query
word in a partition. The service remembers these amounts.

On 2026-10-06, a homelab A/B asked 10 new queries in each setting. Without the
sample, the queries took 2.64 s instead of 4.12 s and found as many documents.

## Decision

1. When every query word has a remembered amount, the rarest word leads. Steps 2
   to 6 of ADR 7 apply.
2. When a query word has no remembered amount, ask every word and compound word in
   all partitions at once, without `urls`.
3. Remember the amounts that the spread measured, so that a later query with the
   same words has a leading word.

## Consequences

* A query with a word that has no remembered amount asks the peers once.
* That query does not narrow the other words to the documents of the rarest word.
  A later query with the same words does.
* A word that no peer counts gets no remembered amount. A query with that word
  always asks every word at once.
