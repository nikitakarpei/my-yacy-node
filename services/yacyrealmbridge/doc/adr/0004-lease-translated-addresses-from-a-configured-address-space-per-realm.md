# 4. Lease translated addresses from a configured address space per realm

Date: 2026-09-14

## Status

Accepted

## Context

The two realms give the bridge different room. One gives an address block the bridge answers
on in full; the other gives one address and its ports. A translated address must stay with its
hash for the lease, whatever the peer does with its native address.

## Decision

Each realm has an operator-configured translated address space: the addresses of one prefix, or
the ports of one host. A lease takes one address from that space once per hash, by one atomic
step every instance shares, and gives it back when the lease expires.

Deriving the translated address from the native address, with no state, was considered and
rejected: a peer that moves would change its translated address, which the lease forbids.

## Consequences

Both directions follow one rule. A prefix space is as good as unbounded; a port space is not, so
translation into the one-address realm can run out, and the bridge then translates no further
peers into it. A translated seed reaches its peer at one address only.
