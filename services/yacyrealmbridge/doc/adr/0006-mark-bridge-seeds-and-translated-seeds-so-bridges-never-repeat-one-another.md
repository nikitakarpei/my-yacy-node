# 6. Mark bridge seeds and translated seeds so bridges never repeat one another

Date: 2026-09-14

## Status

Accepted

## Context

Several independent bridges over the same two realms each see every peer. Each would give one
native peer its own translated address, and gossip would carry all of them into one native
peer's roster. In a realm that treats an address as identity, one hash at several addresses is
a conflict, not a redundancy. The bridges have no channel to each other and must not need one.

## Decision

A bridge's own seed in each realm carries a plain bridge mark in its tags, and no bridge
translates a seed that carries it, whoever set it. Faking the mark costs only the faker its own
translation, so the mark needs no signature.

A public bridge, the default, puts a translation mark in the tags of every translated seed,
stating when it first leased that address for that hash, signed with its key over the seed it
marks. No bridge translates a seed that carries a translation mark, verified or not, so a peer
crosses one public bridge and never a chain. What a mark states counts only under a key the
operator trusts: a bridge does not translate a peer whose hash a trusted translation carries.

When two bridges have translated one hash before seeing each other, the older translation
stands, and the smaller translated address on equal times; the other bridge withdraws its
translation and stops answering on it. Both bridges read the times in the two marks, never their
own clocks, so both reach the same answer. Bridges read the marks that native peers gossip and
never talk to each other.

A private bridge marks no translated seed. It is for a lone deployment, such as a few peers
behind one NAT exposed through their owner's own bridge, where no second bridge will ever
translate the same peers.

## Consequences

One native peer converges on one translated address per realm without coordination, and the
bridge that withdraws is the newer one, which fewer peers have seen. A peer that stops answering
is dropped by the native peers themselves, and any bridge's own peer stays in the realm it
serves. Stock YaCy stores a seed as one map and re-emits every entry of it, so
the marks survive gossip; a seed longer than 16000 characters is rejected whole.

A translation mark under an untrusted key stops that seed and nothing else: the bridge
translates the peer from the other realm anyway. A native seed that states a translated peer's
hash changes nothing either; the peers of that realm resolve the conflict by their own rules.
Two bridges that do not trust each other both translate every peer, and a private bridge's
translations look native to every other bridge, which may translate them again.
