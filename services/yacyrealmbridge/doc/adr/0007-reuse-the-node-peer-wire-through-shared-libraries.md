# 7. Reuse the node's peer wire through shared libraries

Date: 2026-09-14

## Status

Accepted

## Context

`yacynode` already posts one form to one peer address and reads the message back, and already
asks a caller's advertised address whether a YaCy peer of the network answers there. The bridge
needs both unchanged, and both carry no node vocabulary.

## Decision

Both units move from the node into `libraries/`, and the bridge and the node build on them
there. The bridge writes no peer wire code of its own.

## Consequences

One definition of the peer greeting and of the back-ping serves the node and the bridge. The
move changes the node's layout before the bridge's first line of code.
