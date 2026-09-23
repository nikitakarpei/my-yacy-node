# 7. Choose the leading word from a fresh sample

Date: 2026-09-23

## Status

Proposed

## Context

A query of n words and c compound words asks 16 × (n + c) word partitions in discovery.
Discovery of all words serves one choice: the leading word, whose documents are the
candidates of the join. A document joins only in the partition of its URL hash, so a
different word asked in a partition without candidates adds no document.

The cross-check asks, with `urls`, the partitions that discovery did not complete. Real
YaCy peers ignore `urls`, so peer judgements skip them for a retrial interval. Its main
use is the 1000-document abstract cap of `yacynode`.

The judged queries of 2026-09-21 give a median of 3942 documents for the rarest word. At
that size almost every partition has candidates, and the saving of discovery asks is
between 0.2 % and 3.5 %.

## Decision

Discovery asks every query word in one partition that it chooses at random for each query.
A word with a count and a complete abstract gets a sample: its documents in that
partition. The rarest word whose sample predicts a saving above a configured minimum leads.
A word without a sample cannot lead. A peer can make a word look common, which removes the
saving, but a silent peer cannot make a word look rare.

The leading word and every compound word are asked in all partitions. The other words are
asked once in each partition that has candidates, with the candidates in `urls`, as soon
as the leading word's answer of that partition arrives. A partition with more candidates
than `YACYDHTSEARCH_DOCUMENTS_TO_MATCH_CEILING` is asked without `urls`.

A partition where the answer of the leading word is incomplete is asked for all words
without `urls`. The local join filters every answer, so an answer that ignored `urls` is
used as it is.

Discovery is one round in one open run of replica asks. The spread has two rounds:
discovery and URL metadata. URL metadata is asked only for joined documents that no
discovery answer carried metadata for.

Counts that peers claim can choose which peers a query asks. Each value that the plan uses
is a fresh answer, in the same query, from a peer that the full plan asks for the same
word partition. No count, sample or hint is kept between queries.

The new discovery runs first in shadow beside the full discovery, which includes the
cross-check. It is switched on when the shadow shows no more calls than today, join
equality where the leading word is complete, and almost no joined documents found only by
the cross-check.

Before the switch, `yacynode` replaces its 1000-document abstract cap with a byte budget
for each answer. After the switch, the cross-check round, peer judgements, their bucket
and `YACYDHTSEARCH_CROSS_CHECK_RETRIAL_INTERVAL` are removed.

## Consequences

The spread loses a round, the peer judgements and their state. The number of calls stays
at or below today, and a query saves calls only when its rarest word has fewer than about
50 documents. The sample adds one step before the leading word.

A partition where every replica failed stays without an answer for that query. A partition
with more candidates than the ceiling gets a full abstract, which is larger but loses no
document. Rarity for ranking comes from fewer partitions for the words that are not
leading.

A stored hint may later name the word to ask first and replace the sample. The saving stays
limited by how rare the leading word is.
