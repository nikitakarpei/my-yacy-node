# 4. Choose the translated address scheme per realm

Date: 2026-09-14

## Status

Accepted

## Context

The two realms give the bridge different room. One gives an address block the bridge answers
on in full; the other gives one address and its ports. One scheme for both either wastes what
the block gives or does not fit into one address.

## Decision

Each realm has its own translated address scheme behind one contract: the translated address of
a native address, and the native address behind a translated one. A realm that gives an address
block derives the translated address from the native address and keeps no state. A realm that
gives one address allocates a port from a bounded pool once per registration and releases it
when the registration lapses.

## Consequences

A translated address stays the same while the native address holds its registration. In a block
realm any instance of the bridge decodes it without state. A pool realm can run out, and the
bridge then refuses new registrations for it. A translated seed offers plain HTTP at one
address only.
