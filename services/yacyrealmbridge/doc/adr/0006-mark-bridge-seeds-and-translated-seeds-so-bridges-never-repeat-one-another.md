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

Every translated seed carries a translation mark in its tags that names the realm the seed came
from, signed with the bridge's key over the seed it marks. A bridge reads only translation marks
signed by keys its operator trusts, and drops a seed under a mark it cannot verify. A trusted
translation never goes back into the realm its mark names and goes onward into any other realm.

A bridge does not translate a peer whose hash a trusted translation in the receiving realm
already carries; a native seed of that hash there withholds nothing.

When two bridges have translated one hash before seeing each other, the smaller translated
address stands, and the other bridge withdraws its translation and stops answering on it.
Bridges read the marks that native peers gossip and never talk to each other.

## Consequences

One native peer converges on one translated address per realm without coordination, a peer
that stops answering is dropped by the native peers themselves, and any bridge's own peer stays
in the realm it serves.

A further realm can be bridged onto one that is bridged already, through bridges whose
operators trust each other and share realm names. Stock YaCy stores a seed as one
map and re-emits every entry of it, so the marks survive gossip; a seed longer than 16000
characters is rejected whole.

A rogue translation mark under an untrusted key changes nothing: the bridge translates the peer
anyway and drops the marked seed. A native seed that states a translated peer's hash changes
nothing either; the peers of that realm resolve the conflict by their own rules. Two bridges that
do not trust each other both translate every peer. Admission processes and trust chains between
operators are future work.
