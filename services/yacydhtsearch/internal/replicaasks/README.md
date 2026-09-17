# Replica asks

## Why this package exists

A YaCy network stores each word partition on several peers, its replicas. A
search used to ask every replica of every word partition at once and wait for
the slowest one. With a hundred peer calls per search, the slowest call is
almost always a slow one, so every search paid the tail latency of the network.
The replicas hold the same postings, so most of those calls added nothing.

This package asks the replicas of one word partition in turn. The first `n`
listings cover the partition and the other calls are cancelled. A replica that
lists nothing or fails is replaced at once. A replica that stays silent past a
hedge delay gets a second replica asked beside it. Coverage ends the asks
instead of the slowest peer.

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
