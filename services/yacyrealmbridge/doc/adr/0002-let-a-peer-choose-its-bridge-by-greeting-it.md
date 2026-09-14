# 2. Let a peer choose its bridge by greeting it

Date: 2026-09-14

## Status

Accepted

## Context

A bridge that finds peers from seed lists and announces translated seeds repeats rumours it
cannot confirm. Several such bridges over the same two realms race to represent one peer, and a
realm that treats an address as identity cannot hold a peer that several bridges represent at
once.

## Decision

The bridge is a YaCy peer of its own in each realm and publishes a seed list that holds only
that peer. A peer registers by adding the seed list and greeting the bridge. The bridge confirms
the advertised address the way any YaCy peer does, and holds the registration for a lease that
each greeting renews. The hello answers of the bridge's peer list the registered peers of the
other realm; that is the whole announcement.

## Consequences

The bridge discovers nothing and never crawls a realm. A peer is represented in the other realm
by exactly the bridges it chose, so anyone may run a bridge without racing another one. A peer
that stops greeting is withdrawn after the lease, and nothing else keeps liveness. A peer that
never adds a bridge's seed list stays invisible across the realm boundary.
