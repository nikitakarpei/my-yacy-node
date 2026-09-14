# 3. Identify a registered peer by its native address

Date: 2026-09-14

## Status

Accepted

## Context

A peer hash is stated by whoever sends a seed, and nothing in the peer protocol proves it. A
bridge that keys registrations by hash lets a rogue take a translated address away from the
real peer.

## Decision

A registration and its translated address belong to the native address that greeted the
bridge. The hash is payload. It decides two things only: a peer whose hash is registered
natively in the receiving realm gets no translated address there, and the bridge's own seeds
do not cross.

## Consequences

A peer that moves registers again from its new address and gets a new translated address; the
old one lapses with its lease. A registration is as strong as the address the realm gives. A
rogue hash is forwarded as stated; proof of hash ownership is not the bridge's concern.
