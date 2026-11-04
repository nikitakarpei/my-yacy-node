# 27. Admit senior peers by committed identities

Date: 2026-11-04

## Status

Proposed. Not built. The user decides the open points below before it is built.

## Context

A party can make identities until one hash lands next to the position of a word. That party then
holds the postings of the word in that partition and answers its searches. One identity costs one
work puzzle (`yacymodel` ADR 0001), so one slot costs seconds of GPU time.

A quota, a fee, an invite or an uptime rule limits how many identities a party gets. It does not
limit where they sit, because the party makes keys offline and admits only the lucky ones. A
position must come from randomness that nobody knows when the party commits to its key.

## Decision

The node protects its own data first. These rules need no admission:

- The node purges a word partition only when its position is outside a distance from the word
  position. More peers near the word do not move this limit.
- Only peers with a signed seed count as holders for a purge.
- An empty answer settles a partition in a search only when a second replica agrees.
- The node chooses the replicas of a word from different autonomous systems.

A new Go identity commits before it gets its position:

1. The peer makes its key and appends a commit with its key digest and a drand round `R` to an
   admission log. The commit counts against a quota of its autonomous system and address prefix.
2. Witnesses cosign a checkpoint that holds the commit before round `R`.
3. The nonce is the first nonce at or after `S = SHA-256(randomness of R ‖ key digest)` that
   solves the puzzle. One commit gives one position.
4. The peer serves the commit, the proof and round `R` on a peer endpoint.

A Go peer checks the proof once for each peer and keeps the result. A peer that shows an earlier
solving nonce proves a false identity. A committed peer becomes a DHT target after a probation of
reachability, and it renews its admission each epoch.

Go peers send postings to peers without admission, but those peers do not settle a search or start
a purge.

## Open points

- Who runs the logs and the witnesses, and who signs the default list of them.
- The quota for each autonomous system and prefix, the probation time and the epoch.
- Whether drand is a hard dependency for a new identity.
- Whether a Go-only position that changes at each renewal must also limit the identities that a
  party keeps. That position stores more copies.

## Considered alternatives

More work for each identity. Rejected, because a rented GPU does the work of a Pi for less than
the cost of the power of the Pi.

A quota, a fee, an invite or a token alone. Rejected, because each limits the count of identities,
not their positions.

A position from the IP address, as BitTorrent BEP 42 does. Rejected, because a peer with a dynamic
address moves on the ring and loses its data.

Detection of peers that sit unusually near a word. Rejected as a defence, because it does not see
one peer in one slot.

## Consequences

To take one slot of a word, a party needs about N/k commits, where N is the count of admitted
peers and k is the redundancy. Each commit uses a quota. The log shows the commits to all
operators.

Java peers do not check commits, so their DHT traffic stays open to chosen positions. The
protection grows with the number of honest Go peers. Positions do not change, so the quota limits
the rate of lucky identities, not their total.

A new peer waits for one checkpoint and one round, and then for the probation. Each Go peer scans
about 2^28 hashes once to check a new peer. Existing Go peers make new keys before the rule starts.
