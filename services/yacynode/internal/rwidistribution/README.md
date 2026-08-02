# RWI distribution

This package offers this node's stored RWI postings to the peers the DHT makes
responsible for them. It deletes a posting when enough closer peers hold it.

## Behavior

A cycle offers nothing while the node knows fewer reachable peers than the
configured minimum. Above that minimum, each cycle takes the due postings and
offers each one to its responsible peers. A newly stored posting is due at once.
When the DHT makes this node responsible, this node holds one of the replicas. A
cycle records all of its results or none of them.

A peer accepts a posting in two steps. It first accepts the posting and names the
URLs it does not know. This node then sends the metadata for those URLs. The
posting counts toward redundancy only after both steps reach the peer. A peer
that stops accepting a remote index gets no more offers. The postings it holds
keep counting for it.

Each cycle sets when the postings it offered are next due. A posting that
reached its redundancy is next due after the longest offer interval, and its
holders get it again. A posting that fell short is next due after a shorter
interval, which starts at the shortest, doubles on each further shortfall up to
the longest, and returns to the shortest when redundancy is met.

A peer that declines an offer gets no new replicas, and the postings it holds are
still offered to it. When it asks for a pause, the posting waits that pause if
the pause is longer than the interval.

The node deletes a posting when the redundancy is reached in holders closer to
the posting's DHT position than this node. A holder counts only while the node
can reach it. A deleted posting leaves the local index, the work queue, and the
replica record. Its URL metadata stays until storage eviction reclaims it.

A holder goes stale when closer peers displace it, or when the peer stays
uncontacted past the peer roster's credibility window. A stale holder leaves the
replica record. The posting is then offered to as many further peers as it needs
replicas. An uncontacted peer gets the posting again if contact resumes. A
displaced peer does not.

## Divergence from YaCy

YaCy removes a posting from local storage when it selects that posting for
transfer. This node deletes a posting only after enough closer peers hold it.

YaCy keeps no record of which peer holds which posting. This node keeps the
record and offers the posting again when a replica is lost. YaCy takes work from
a random range of the word hash space. This node takes work from a due-time
schedule.
