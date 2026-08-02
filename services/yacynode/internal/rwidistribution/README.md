# RWI distribution

This package offers this node's stored RWI postings to the peers the DHT
makes responsible for them, and keeps offering until a configured number of
those peers have accepted. A posting stays on this node regardless of how
many peers accept it; distribution only replicates, it never deletes.

## Behavior

A cycle offers nothing while the node knows fewer than the configured
minimum of reachable peers.

A newly stored posting is due for an offer immediately. Each cycle offers
the due postings to their responsible peers.

A posting that reaches its redundancy is offered again after the refresh
interval.

A posting that misses its redundancy is offered again after its retry wait,
or after the peer's requested delay, whichever is later. The retry wait
starts at the retry interval and doubles on every further miss, up to the
refresh interval. The retry wait goes back to the retry interval when the
posting reaches its redundancy.

A peer accepts a posting in two steps. It first accepts the posting and
names the URLs it does not recognize. This node then sends metadata for
those URLs. The posting counts toward redundancy only after both steps
reach the peer.

An acceptance stops counting when closer peers displace the peer that gave
it, or when the peer stays uncontacted past the peer roster's credibility
window. The posting is then offered to as many further peers as it needs
replicas.

A displaced peer is not offered the posting again. A peer uncontacted past
the credibility window is offered the posting again if contact resumes.

A peer that stops accepting a remote index receives no further offers. The
postings it already holds keep counting for it.

A cycle either records all of its results or none of them. A cycle that
records none of them offers the same postings again.

Deleting a posting from local storage also removes it from the distribution
work queue; a deleted posting is never offered.
