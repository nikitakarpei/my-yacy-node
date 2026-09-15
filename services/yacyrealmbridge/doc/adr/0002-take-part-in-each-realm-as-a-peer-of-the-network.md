# 2. Take part in each realm as a peer of the network

Date: 2026-09-14

## Status

Accepted

## Context

The second realm wants to be part of the network that already lives in the first. The bridge
must know the peers of both realms, and the peers of both must know the bridge. A bridge that
confirms only the peers that explicitly signed up with it keeps most of the existing realm out.

## Decision

The bridge is a YaCy peer of its own in each realm. It learns that realm's peers the way any
peer does: from seed lists, from the peers that greet it, and from the seeds carried in answers.
It confirms a peer the way any YaCy peer does, keeps confirming, and drops a peer that stops
answering. The hello answers of its peer in a realm list the confirmed peers of the
other realm; that is the whole announcement.

## Consequences

Nothing registers and nothing leases: the bridge's own view of a realm is the one source of
which peers it translates. Several bridges over the same two realms each see every peer, and
each would translate it; a later decision keeps them from repeating one another.
