# 15. Hand off postings to closer peers

Date: 2026-08-02

## Status

Accepted

Amends ADR 0008.

## Context

The node kept every posting it stored and offered each one again at every refresh interval. That
traffic grows with the store and never falls.

Most of the store lies outside the word-hash range the node is responsible for. A DHT search never
routes a query for that range to this node, so those postings answer no one and hold quota.

ADR 0008 planned redistribute-then-delete as an eviction policy, driven by quota pressure, at URL
granularity. Outbound distribution was a non-goal then. The node now records which peer accepted
which posting, so redundancy is a standing fact rather than something to establish under pressure.

Upstream YaCy deletes a posting when it transfers it, so its storage is shared across the network.

## Decision

Delete a posting during the distribution cycle once at least `YACY_DISTRIBUTION_REDUNDANCY` peers
that hold the posting are strictly closer to the posting's DHT position than this node.

A peer holds the posting if it accepted the posting in an earlier cycle and the node can still reach
it, or if it accepts the posting in the current cycle.

This excludes the credibility-window grace that redundancy accounting gives an unreachable peer.
That peer still counts toward redundancy, but not toward handing the posting off.

The distribution cycle owns this deletion. It deletes through the same storage primitive as quota
eviction, in the transaction that records the cycle, so the deletion is atomic with the cycle result.

This node counts as one of the responsible peers, so peers owe one replica fewer when the node is
inside the responsibility window. That also bounds the rule: inside the window, fewer peers than the
redundancy are closer, so the deletion cannot fire.

ADR 0008's redistribute policy is dropped. Eviction stays quota-driven and deletes without
transferring.

No new configuration variable. `YACY_DISTRIBUTION_ENABLED`, off by default, is the consent to hand
postings away. `YACY_DISTRIBUTION_REDUNDANCY` is the dial. A deployment that already enabled
distribution loses the postings it holds outside its responsibility range, once closer peers hold them.

## Considered alternatives

Deleting once redundancy is met, without the closer-than-this-node test, was rejected. Every node would
then forward and keep nothing, so each hop multiplies copies.

Deleting whatever the node is not responsible for, without counting holders, was rejected. Two nodes
with different roster views would each delete after handing the posting to the other.

Keeping ADR 0008's eviction-time redistribution was rejected. It establishes under memory pressure, at
URL granularity, a fact the node already holds per posting.

## Known limitations

This node cannot repair a handed-off posting if its holders lose it.

Two nodes whose rosters disagree about who is closer can pass a posting back and forth. The
strictly-closer test bounds this but does not remove it.

A URL whose last posting is handed off keeps its metadata row until the quota sweeper reclaims it.

## Consequences

The store converges on the range the node is responsible for, where its postings answer searches and
a lost replica is still repaired. Refresh traffic falls with the store. Storage is shared, as in
YaCy. Operators observe handoffs and a falling backlog of scheduled postings through metrics.
