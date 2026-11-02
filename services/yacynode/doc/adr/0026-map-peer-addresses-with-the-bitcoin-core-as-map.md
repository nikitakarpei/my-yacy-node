# 26. Map peer addresses with the Bitcoin Core AS map

Date: 2026-11-02

## Status

Accepted

## Context

The operator must see how the reachable peers spread across the autonomous systems of their
addresses. When one hosting provider holds most of the peers, that provider can hide the rest of
the network from the node. The node needs a map from an IP address to the number of its
autonomous system, without a network call and without an account at a data provider.

## Decision

Embed the AS map that Bitcoin Core ships in `src/node/data/ip_asn.dat`, under its MIT license,
and read it with a port of the Bitcoin Core interpreter. The file `bitcoin-core-tag` beside the
map sets the Bitcoin Core release. `make refresh-asmap` gets the map of that release.

## Considered alternatives

An address range of fixed length, as Bitcoin Core uses without a map. Rejected because one large
provider spans many ranges, so the range count hides the concentration that the panel must show.

A map that the operator supplies in configuration. Rejected because every operator must then get
and replace a file before the panels show data.

A database from a commercial provider. Rejected because it needs an account and a license key.

## Consequences

The binary is larger by the size of the map. The map shows the state of the Bitcoin Core release,
and it ages until a maintainer changes the tag and gets the map again.

The map fills unassigned IPv4 space with the autonomous system of an adjacent range. A peer on
an IPv4 range that moved to a new owner shows under the previous owner until the map changes.
