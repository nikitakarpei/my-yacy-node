# 23. Take the peer key from configuration and bind peer identities

Date: 2026-10-28

## Status

Accepted

## Context

`yacymodel` ADRs 0002 and 0004 derive the peer hash from a peer key and sign the address
fields of a seed. The node keeps its peer hash for the life of its data, because its postings
belong to that position on the DHT ring.

## Decision

The node reads its peer key from `YACY_PEER_KEY` and stores no secret. The node does not start
without a valid key. `YACY_INITIAL_PEER_HASH` is removed.

- A deployment holds secrets in its environment; the data directory holds none. `yacypeer` makes
  and keeps the key for a one-step peer.
- The node stores the peer hash that derives from the key, and refuses to start when the stored hash
  is different. A different key would serve postings for another ring position.

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

- Seeds without a valid identity stay accepted, because Java peers and pruned gossip copies have
  none. Removal of the columns gives no more than this.
- After binding, only the key holder can change the peer's address, and an older address cannot
  return.

The roster observer reports each drop at WARN with its reason.

## Consequences

A peer that the roster has not bound has only the protection of a legacy peer.

A peer that the roster drops from its known peers loses its binding.

Requests stay unsigned, so `iam` is still a claim. Java peers still accept forged seeds.
