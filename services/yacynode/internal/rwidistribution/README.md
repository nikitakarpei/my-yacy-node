# RWI distribution

This package offers this node's stored RWI postings to the peers the DHT makes
responsible for them, and keeps offering until a configured number of those
peers have accepted. A posting stays on this node regardless of how many peers
accept it. Distribution only replicates; it never deletes.

## Behavior

A cycle offers nothing while the node knows fewer than the configured minimum
of reachable peers. Above that minimum, each cycle takes the postings that are
due and offers each one to its responsible peers. A newly stored posting is due
immediately. Deleting a posting from local storage also removes it from the
work queue, so a deleted posting is never offered.

A peer accepts a posting in two steps. It first accepts the posting and names
the URLs it does not recognize. This node then sends metadata for those URLs.
The posting counts toward redundancy only after both steps reach the peer. A
peer that stops accepting a remote index receives no further offers, but the
postings it already holds keep counting for it.

A posting that reaches its redundancy is offered again after the refresh
interval, to the peers that hold it. The offer restores the replica if the peer
no longer has it. A peer that does not accept an offer is passed over when new
replicas are placed, but it still receives these refresh offers. A peer that
does not accept the refresh keeps its place, and the posting waits the refresh
interval again before the next offer.

A posting that misses its redundancy is offered again after its retry wait, or
after the peer's requested delay, whichever is later. The retry wait starts at
the retry interval and doubles on every further miss, up to the refresh
interval. The retry wait goes back to the retry interval when the posting
reaches its redundancy.

An acceptance stops counting when closer peers displace the peer that gave it,
or when the peer stays uncontacted past the peer roster's credibility window.
The posting is then offered to as many further peers as it needs replicas. A
displaced peer is not offered the posting again. A peer uncontacted past the
credibility window is offered the posting again if contact resumes.

A cycle either records all of its results or none of them. A cycle that records
none of them offers the same postings again.

## Divergence from YaCy

YaCy computes the same DHT position for a posting and selects the same
responsible peers, so placement matches and the wire exchange is the same two
steps. The two systems differ in what happens after a peer accepts.

YaCy removes a posting from local storage when it selects that posting for
transfer, and puts it back only if the transfer fails. Storage is therefore
shared across the network, and a transferred posting has no further cost to the
peer that sent it. YaCy keeps no record of which peer holds which posting, so it
does not detect the loss of a replica and does not repair one. It selects work
from a random range of the word hash space each round, which needs no schedule
because a transferred posting leaves the pool.

This node keeps every posting it stores. Storage is therefore not shared, and
the node carries the full cost of its own index. In exchange, the node holds a
record of which peer accepted which posting, detects a replica that the DHT no
longer justifies, and offers the posting again to close the gap. Work comes
from a due-time schedule rather than a random range, because the pool never
shrinks. The node also pays a constant background cost: every posting is
offered again once per refresh interval for as long as it is stored.
