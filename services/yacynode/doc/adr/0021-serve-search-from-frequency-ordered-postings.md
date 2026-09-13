# 21. Serve search from frequency-ordered postings

Date: 2026-09-13

## Status

Proposed

## Context

When a peer searches, the node reads every posting of each search word. It keeps the 1000
postings with the most hits and throws the rest away. For each excluded word it reads every
posting into memory, with no limit at all. So the work for one search depends on how many
postings a word has, not on what the peer asked for. A common word makes the node read its
whole list to return ten results.

The node stores postings by word and URL. Nothing keeps a word's postings in order of hits, so
the only way to find the best ones is to read all of them. The node already keeps other lists
next to the posting store and updates them each time a posting is stored or purged, for example
the list of words that point to a URL.

## Decision

Keep two more lists next to the posting store, each in its own unit, and update them in the same
transaction that stores or purges a posting:

- for each word, its postings in order of hits, most hits first;
- for each word, how many postings the node holds.

A search reads the postings of a word in that order and decides by itself when to stop. It first
reads the counts and starts from the word with the fewest postings. For each candidate document
it looks up the other words, the filters, and the excluded words by key. It stops when the best
documents it found score at least as well as any document it has not read yet. The order of the
list makes that check possible. The count of a word answers the `indexcount` field with no read
of the postings.

The node still stops after a fixed number of postings per word, in case the check never lets it
stop, and it still stops at the request deadline. When either limit stops a search, the node
reports it through metrics.

A store from before this decision starts with empty lists. The node does not migrate it. An
operator recreates the store, as ADR 0017 did for the key format.

## Considered alternatives

Start from the rarest word without an ordered list. Rejected: the node still reads the whole
rarest word, so this bounds memory but not disk reads.

Only cap the number of results. Rejected: the request already caps the results, and the cost is
in what the node reads to choose them.

One ordered list per filter, such as language or site. Deferred: it multiplies the writes and
the storage of every posting. It is added once metrics show that filtered searches return too
few results while the word holds more that match.

Lists in document order with skip pointers and block maxima, as Lucene uses. Rejected: the
storage engine offers no seek and no block maxima, and the index is too small to need them.

Drop postings with few hits when they arrive. Rejected: it loses data and changes what
`indexcount` means to a peer.

## Known limitations

A strict filter can return fewer results than the node holds, because the node applies the
filter only to the postings it examines before the fixed limit.

The stop check compares the sum of hits only. When two documents have the same sum, the node
orders them by term spread and URL hash. A search where all postings have the same hits runs to
the fixed limit.

Each posting costs two more writes and the storage of two more keys.

## Consequences

The reads of one search are bounded by the request and by the rarest search word, never by the
size of a common word. A search with one rare word costs a few dozen reads. Excluded words no
longer load a whole list into memory. Operators see through metrics when a search stopped at a
limit instead of on its own check.
