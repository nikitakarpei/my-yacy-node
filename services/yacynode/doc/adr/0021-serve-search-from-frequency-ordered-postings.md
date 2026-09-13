# 21. Serve search from frequency-ordered postings

Date: 2026-09-13

## Status

Proposed

## Context

A search reads every posting of each query term, keeps the 1000 most frequent, and counts the
rest. It reads every posting of each excluded term into memory with no bound at all. The work of
one request grows with the size of the word, not with the request. A common word makes the node
read its whole posting list for a query that returns ten results.

The posting store keys postings by word and URL. Nothing orders a word's postings by relevance,
so the only way to find the most relevant ones is to read them all. The node already keeps
derived views of the posting store through posting observers, such as the words that reference a
URL and the offer schedule of a posting.

## Decision

Keep two more derived views of the posting store, each in its own unit and kept by a posting
observer in the transaction that stores or purges the posting:

- the postings of each word ordered most frequent first;
- the amount of postings the node holds for each word.

Search reads a word's postings in that order and stops on its own evidence. It reads the amounts
first and drives the join from the term with the fewest postings. It looks up the other terms,
the filters, and the excluded terms per candidate by key. It stops when the best documents found
score at least as well as any document it has not read yet, which the frequency order lets it
prove. The amount of a word answers the `indexcount` field without a read of the postings.

A fixed limit of postings examined per term stays as the safety bound for the case the proof
never closes, and the request deadline stays as the second bound. When either bound fires, the
node reports it through metrics.

Existing stores start with empty views. The node does not migrate a store from before this
decision; an operator recreates the store, as ADR 0017 did for the key format.

## Considered alternatives

Pivot the join on the rarest term without an ordered view. Rejected: it bounds memory, not disk,
because the rarest term is still read in full.

Keep a cap on results only. Rejected: the results a peer asks for are already capped by the
request, and the cost lies in what the node reads to choose them.

One ordered view per filter facet, such as language or site. Deferred: it multiplies the writes
and the storage of every posting. It is admitted once metrics show filtered searches that
under-fill while the word holds more matching postings.

Document-ordered lists with skip pointers and block maxima, as Lucene uses. Rejected: the vault
exposes no seek and no block maxima, and the index size does not justify them.

Drop low-frequency postings at write time. Rejected: it is lossy and changes what `indexcount`
means to a peer.

## Known limitations

A selective filter can return fewer results than the node holds, because the filter is applied
inside the postings the safety bound lets the node examine.

The stop rule proves the order on the sum of hits only. Ties on that sum are broken by term
spread and URL hash, so a query whose postings all carry equal hits runs to the safety bound.

Every posting costs two more writes and the storage of two more keys.

## Consequences

The reads of one search are bounded by the request and by the rarest query term, never by the
size of a common word. A query with one rare term costs a few dozen reads. Excluded terms no
longer load a word into memory. Operators see through metrics when a search stopped on a bound
instead of on proof.
