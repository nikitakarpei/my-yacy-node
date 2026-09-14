# 6. Label translated seeds so bridges never repeat one another

Date: 2026-09-14

## Status

Accepted

## Context

Several independent bridges over the same two realms each see every peer. Each would give one
native peer its own translated address, and gossip would carry all of them into one native
peer's roster. In a realm that treats an address as identity, one hash at several addresses is
a conflict, not a redundancy. The bridges have no channel to each other and must not need one.

## Decision

Every seed a bridge emits carries a bridge label in the seed's tags, signed with the bridge's
key over the seed it marks: the translated seeds and the bridge's own seed in each realm alike.
A bridge reads only labels signed by keys its operator trusts; any other label is no label. A
labelled seed is never a native peer: no bridge translates it onward, and no bridge carries
another bridge's own peer into the other realm.

A bridge does not translate a native peer whose hash a labelled seed in the receiving realm
already carries.

When two bridges have translated one peer before seeing each other, the smaller translated
address stands, and the other bridge withdraws its translation and stops answering on it.
Bridges read the labels that native peers gossip and never talk to each other.

## Consequences

One native peer converges on one translated address per realm without coordination, a peer
that stops answering is dropped by the native peers themselves, and a trusted bridge's own peer
stays in the realm it serves. Stock YaCy stores a seed as one
map and re-emits every entry of it, so the label survives gossip; a seed longer than 16000
characters is rejected whole.

A rogue label under an untrusted key changes nothing: the bridge translates the peer anyway. A
seed under an untrusted label reads as a native peer, so an untrusted bridge's translations can
be translated onward, and two bridges that do not trust each other both translate every peer.
Admission processes and trust chains between operators are future work.
