# RWI distribution

This package offers this node's stored RWI postings to the peers the DHT
makes responsible for them, and keeps offering until a configured number of
those peers have accepted. A posting stays on this node regardless of how
many peers accept it; distribution only replicates, it never deletes.

## Behavior

A cycle offers nothing while the node knows fewer than the configured
minimum of reachable peers. This keeps a thin roster, such as right after
startup, from being read as "no peer is responsible", which would drop
replica ledger entries for peers that are still holding a posting.

A newly stored posting becomes due for an offer immediately. Each cycle,
due postings are offered to their responsible peers. A posting that reaches
its configured redundancy is left alone until the next refresh interval, so
its replication is checked again as the peer set changes. A posting that
fails to reach redundancy — no responsible peer accepted it, no peer was
found responsible, or a peer asked to be retried later — becomes due again
after a retry interval or the peer's own requested delay, whichever is
later.

If a peer that previously accepted a posting later leaves the network or is
no longer responsible for it, that acceptance no longer counts, and the
posting becomes eligible for offering again until redundancy is restored.

Deleting a posting from local storage also removes it from the distribution
work queue; a deleted posting is never offered.
