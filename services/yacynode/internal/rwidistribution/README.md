# RWI distribution

This package offers this node's stored RWI postings to the peers the DHT
makes responsible for them, and keeps offering until a configured number of
those peers have accepted. A posting stays on this node regardless of how
many peers accept it; distribution only replicates, it never deletes.

## Behavior

A cycle offers nothing while the node knows fewer than the configured
minimum of reachable peers. This avoids offer traffic into a roster too
thin to be worth offering into, such as right after startup.

A newly stored posting becomes due for an offer immediately. Each cycle,
due postings are offered to their responsible peers. A posting that reaches
its configured redundancy is left alone until the next refresh interval, so
its replication is checked again as the peer set changes. A posting that
fails to reach redundancy — no responsible peer accepted it, no peer was
found responsible, or a peer asked to be retried later — becomes due again
after a retry interval or the peer's own requested delay, whichever is
later.

A peer accepts a posting in two steps. It first accepts the posting itself
and names any URL it does not recognize; this node then sends metadata for
those URLs. A posting counts toward redundancy only once both steps reach
the peer. If the peer already knows every URL, or the metadata step
succeeds, the posting counts immediately. If the metadata step fails, the
posting is treated as not replicated and is offered again at the retry
interval.

An acceptance stops counting when closer peers displace the peer that gave
it, when contact with that peer fails, or when the peer stays uncontacted
past the peer roster's credibility window. The posting is then offered
again, to as many peers as it still needs copies, until redundancy is
restored.

A peer that stops accepting a remote index receives no further offers. The
postings it already holds keep counting for it.

Deleting a posting from local storage also removes it from the distribution
work queue; a deleted posting is never offered.
