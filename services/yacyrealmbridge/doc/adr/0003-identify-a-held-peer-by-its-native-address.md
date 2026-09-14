# 3. Identify a held peer by its native address

Date: 2026-09-14

## Status

Accepted

## Context

A peer hash is stated by whoever sends a seed, and nothing in the peer protocol proves it. A
bridge that keys its view by hash lets a rogue take a translated address away from the real
peer.

## Decision

A held peer and its translated address belong to the native address the bridge confirmed. The
hash is payload. It decides one thing only: a peer whose hash is held natively in the receiving
realm gets no translated address there.

## Consequences

A peer that moves is confirmed at its new address and gets a new translated address; the old
one drops when it stops answering. A held peer is as strong as the address the realm gives. A
rogue hash is forwarded as stated; proof of hash ownership is not the bridge's concern.
