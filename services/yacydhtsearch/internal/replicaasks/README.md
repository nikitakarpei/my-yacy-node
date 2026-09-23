# Replica asks

## Why this package exists

A YaCy network stores each word partition on several peers, its replicas. A
search used to ask every replica of every word partition at once and wait for
the slowest one. With a hundred peer calls per search, the slowest call is
almost always a slow one, so every search paid the tail latency of the network.
The replicas hold the same postings, so most of those calls added nothing.

This package asks the replicas of one word partition in turn. The first `n`
answers that cover the partition settle it, and the other calls are cancelled.
An answer covers when it lists, counts or matches a document of the word, and
lists no document outside the documents to match. A replica that does not cover,
fails, or stays silent past a hedge delay gets the next one asked.

The URL metadata of the documents of one partition is asked the same way. The
replicas are the peers whose abstracts listed a document of the partition, the
peer that listed the most first. One answer that carries the metadata of every
document asked settles the partition.

A peer gets one ask at most. A replica already asked for another word partition
is skipped.

## Prior art

- **Hedged requests** from Dean and Barroso, "The Tail at Scale" (2013). Send
  the same request to a second replica after the first is slow, take the first
  answer, cancel the rest. The hedge delay around the 95th percentile keeps
  the extra load small. This is the hedge rule here.
- **Quorum reads** in Dynamo-style stores. A read is answered once `R` of `N`
  replicas reply. `n` replicas covering a partition is the same knob, with
  `n=1` as the fast path and `n` equal to the redundancy as the join of all
  replicas.
- **Speculative retry** in Cassandra. A replica that has not answered within a
  latency percentile is backed up by another one. Cassandra derives the delay
  from observed latency, which is the later `hedgedelays/answerlatency` here.
- **Retry on empty and on failure** as in gRPC and Envoy retry policies: a
  failed call moves to the next endpoint at once and never waits for the
  hedge delay.
