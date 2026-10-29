# Peer roster

This package owns the set of network peers this node knows, and which of
them it currently considers reachable. It is the single owner of a peer's
reachable status: the announcement cycle confirms or clears it from contact
outcomes, and inbound admission can confirm a caller as reachable too.

## Behavior

Every known peer is written to durable storage, so a restart resumes from
the known roster instead of the seed source. Only the reachable set itself
lives in memory; a restart clears it, and each peer must be reconfirmed
before it counts as reachable again.

A peer becomes known once discovered, from a seedlist or from another
peer's greet response. This node is never known as a peer of itself, even
when a seedlist or a greet response names it. The known roster is bounded;
once it is full, the stalest peer that is not currently reachable is
evicted to make room for a new one.

The roster keeps every host in a peer's seed, its port, and its secure
port. A known peer moves on its own confirmed contact or on a newer signed
seed from discovery. A move takes the peer out of the reachable set until a
contact confirms it again.

The roster binds a peer to the identity of its first signed seed. It then
drops seeds that break the identity, carry an older signature, or are
signed in the future. A peer without a binding is a legacy peer.

The roster also keeps the base URL that answered the last greet or back-ping
of a peer. Deliveries go only to that base URL. A failed greet clears it. The
roster gives out that base URL only while it is one of the base URLs for the
current addresses and identity of the peer.

The reachable set is bounded separately. A peer already marked reachable
always keeps its place when reconfirmed. A newly reachable peer is admitted
only if the reachable set still has room; if it is full, the confirmation
is dropped and logged.

Reachable peers are ranked for gossip: the peer confirmed most recently
ranks first.

Unreachable peers are ranked for probing: a peer reachable most recently
ranks first, so a peer that was reachable right up to a restart is retried
before peers that have never been confirmed. Among peers with no recent
reachable history, the least recently contacted peer ranks first, so
probing rotates through the known roster instead of retrying the same few
peers.

A confirmed reachable peer stays credible for a bounded number of announce
rounds after its last confirmation, and stays credible across a restart. A
failed contact ends its credibility at once.
