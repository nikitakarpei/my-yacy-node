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
key. A translated seed's label names the native address behind it. A labelled seed is never a native peer: no bridge translates
it onward. A bridge does not translate a native peer whose address a label in the receiving realm
already names.

When two bridges have translated one peer before seeing each other, the smaller
translated address stands, and the other bridge withdraws its translation and stops answering
on it. Bridges read the labels that native peers gossip and never talk to each other.

## Consequences

One native peer converges on one translated address per realm without coordination, and a peer
that stops answering is dropped by the native peers themselves. Stock YaCy stores a seed as one
map and re-emits every entry of it, so the label survives gossip; a seed longer than 16000
characters is rejected whole.

The signature proves which key made a label, and nothing yet says which keys to trust: a rogue that gossips a seed labelled as some peer's translation, at a small
address, stops every honest bridge from translating that peer. Bridge admission and trust chains
between operators are future work the signature leaves room for.
