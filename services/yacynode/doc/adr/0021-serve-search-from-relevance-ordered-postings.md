# 21. Serve search from relevance-ordered postings

Date: 2026-09-13

## Status

Proposed

## Context

When a peer searches, the node reads every posting of each search word, keeps the 1000 with
the most hits, and throws the rest away. It reads every posting of each excluded word into
memory with no limit at all. So the work for one search depends on how many postings a word
has, not on what the peer asked for. A common word is read in full to return ten results.

The node stores postings by word and URL. Nothing keeps a word's postings in order of relevance,
so the only way to find the best ones is to read all of them. Relevance today is the hits of the
word alone. The `yacydhtsearch` service ranks with weights tuned on judged queries. There a word
in the title counts many times more than its hits, and a rare query word counts more than a
common one. A posting already carries the title flag, and the node can know how rare a word is.

## Decision

Keep two more lists next to the posting store, each in its own unit, and update them in the same
transaction that stores or purges a posting, as the node already does for the words of a URL:

- for each word, its postings in order of impact, most relevant first. The impact of a posting
  comes from its hits, saturated so that many hits do not dominate, and raised when the word
  appears in the title. It is fixed when the posting is stored;
- for each word, how many postings the node holds.

A search reads the postings of a word in that order and decides by itself when to stop. It first
reads the counts and starts from the word with the fewest postings. For each candidate document
it looks up the other words, the filters, and the excluded words by key. A document scores the
sum of its impacts across the search words, each weighted by how rare the word is in this node.
The search stops when the best documents it found score at least as well as any document it has
not read yet. The count of a word answers the `indexcount` field with no read of the postings.

The node still stops after a fixed number of postings per word, in case the check never lets it
stop, and it still stops at the request deadline. When either limit stops a search, the node
reports it through metrics.

## Considered alternatives

Order by hits alone. Rejected: the tuned weights of `yacydhtsearch` show that the title and the
rarity of a word tell more about relevance than hits, and both are known to the node at no cost.

Rank with the full model of `yacydhtsearch`. Rejected: its phrase and address scores need the
page text or the address, which the node has only after it chose the documents, and that service
applies them to what the node sends anyway.

Start from the rarest word without an ordered list. Rejected: the node still reads the whole
rarest word, so this bounds memory but not disk reads.

One ordered list per filter, such as language or site. Deferred: it multiplies the writes and
the storage of every posting. It is added once metrics show that filtered searches return too
few results while the word holds more that match.

Lists in document order with skip pointers, as Lucene uses, or dropping postings with few hits
on arrival. Rejected: the storage engine offers no seek, and dropping postings changes what
`indexcount` means to a peer.

## Known limitations

A strict filter can return fewer results than the node holds, because the node applies the
filter only to the postings it examines before the fixed limit.

Two documents with the same score are ordered by term spread and URL hash, which the stop check
does not see. A search where all postings have the same impact runs to the fixed limit.

A store from before this decision starts with empty lists, and a change of the impact rule
takes effect only for postings stored after it. The node migrates neither. An operator recreates
the store, as ADR 0017 did for the key format. Each posting costs two more writes and two keys.

## Consequences

The reads of one search are bounded by the request and by the rarest search word, never by the
size of a common word. A search with one rare word costs a few dozen reads. Excluded words no
longer load a whole list into memory. Operators see through metrics when a search stopped at a
limit instead of on its own check.
