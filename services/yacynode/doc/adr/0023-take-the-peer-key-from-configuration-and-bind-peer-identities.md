# 23. Take the peer key from configuration and bind peer identities

Date: 2026-10-28

## Status

Accepted

## Context

`yacymodel` ADRs 0002, 0003 and 0004 derive the peer hash from a peer key and sign the address
fields of a seed. The node keeps its peer hash for the life of its data, because its postings
belong to that position on the DHT ring.

## Decision

The node reads its peer key from `YACY_PEER_KEY` and stores no secret. The node does not start
without a valid key. `YACY_INITIAL_PEER_HASH` is removed.

The node stores the peer hash that derives from the key. It refuses to start when the stored hash is
different, also for a data directory with a hash from before peer keys.

The node puts `Identity` in its seed and signs the address fields each time it makes the seed.

The roster binds the identity of a peer, from the first seed with a valid identity and a valid
address signature:

| Seed | Bound identity | Result |
| --- | --- | --- |
| Valid | None | Bind and accept |
| Valid | Same | Accept; addresses change as before |
| Any | Different | Drop |
| Not valid or not signed | None | Accept as a legacy peer |
| Not valid or not signed | Present | Drop |
| Older address signature | Present | Drop |

The roster observer reports each drop at WARN with its reason.

## Consequences

A peer that the roster has not bound has only the protection of a legacy peer. Removal of the
columns gets legacy treatment.

A peer that the roster drops from its known peers loses its binding.

Requests stay unsigned, so `iam` is still a claim. Java peers still accept forged seeds.
