# 3. Lease one translated address per peer hash

Date: 2026-09-14

## Status

Accepted

## Context

A YaCy peer is known to other peers by its hash. Peers in the receiving realm bind what they
learn to that hash, and a realm that treats an address as identity refuses a hash that shows up
at a new address until its own lease expires. Nothing in the peer protocol proves a hash. That is
the network's known problem inside one realm, and it stays the network's across the bridge.

## Decision

A confirmed peer hash leases one translated address in the other realm for an operator-configured
lease, renewed while the peer stays confirmed. Until the lease expires, that hash has that address
and no other, in both directions. The native address behind it is the one the bridge's view has
for the hash, as any peer's roster does.

## Consequences

A peer that moves inside its realm keeps its translated address, so the receiving realm sees no
change. A peer that goes away frees its address after the lease. A rogue that states another
peer's hash in one realm is forwarded under it into the other, as it would be inside one realm.
A native seed of the same hash in the receiving realm does not withhold the translation; the
peers of that realm resolve the conflict by their own rules.
