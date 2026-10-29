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
port. Discovery writes the addresses of a new peer. For a known peer that is
not reachable, discovery uses only a seed with a valid identity and an
address signature newer than the bound signature. Otherwise, the addresses
of a known peer change only when the peer sends its own seed and is
confirmed reachable.

The roster binds the identity of a peer from the first seed with a valid
identity and a valid address signature. After that, the roster drops each
seed of the peer without that identity and a valid signature, and each seed
with an older address signature for other addresses. An older seed for the
same addresses confirms the peer and keeps the bound signature.

The roster drops each seed signed more than 10 minutes after the local time,
and binds no identity from it. A dropped seed changes nothing. A peer
without a bound identity is a legacy peer. Eviction removes the binding.

The roster also keeps the base URL that answered the last greet or back-ping
of a peer. Deliveries go only to that base URL. A failed greet clears it.
When discovery moves a peer to addresses without that base URL, the roster
clears it.

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
