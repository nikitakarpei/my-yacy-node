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
peer's greet response. The known roster is bounded; once it is full, the
stalest peer that is not currently reachable is evicted to make room for a
new one.

The reachable set is bounded separately. A peer already marked reachable
always keeps its place when reconfirmed. A newly reachable peer is admitted
only if the reachable set still has room; if it is full, the confirmation
is dropped and logged.

Unreachable peers are ranked for probing: a peer reachable most recently
ranks first, so a peer that was reachable right up to a restart is retried
before peers that have never been confirmed. Among peers with no recent
reachable history, the least recently contacted peer ranks first, so
probing rotates through the known roster instead of retrying the same few
peers.

The DHT responsibility query draws only from the reachable set and excludes
this node itself, so a peer is never told it is responsible for its own
postings.
