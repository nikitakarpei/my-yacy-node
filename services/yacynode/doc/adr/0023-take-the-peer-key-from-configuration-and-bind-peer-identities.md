# 23. Take the peer key from configuration and bind peer identities

Date: 2026-10-28

## Status

Accepted

## Context

`yacymodel` ADRs 0001 and 0004 derive the peer hash from a peer key and sign the address
fields of a seed. The node keeps its peer hash for the life of its data, because its postings
belong to that position on the DHT ring.

## Decision

The node reads its peer key from `YACY_PEER_KEY` and keeps no secret on disk.
`YACY_INITIAL_PEER_HASH` is removed. The node stores the hash of the key and does not start with
another key.

The node signs the addresses of its seed. The roster binds a peer to its first valid signed seed:

| Seed | Bound identity | Result |
| --- | --- | --- |
| Valid, same or newer signature | None or same | Accept; bind |
| Valid, older signature | Same | Drop |
| Signed more than 10 minutes ahead | Any | Drop |
| Other identity, or not valid | Present | Drop |
| Not valid | None | Accept as a legacy peer |

Discovery applies the table to every known peer and drops nothing. A move takes a peer out of the
reachable set until a contact confirms it again.

## Consequences

Java peers and pruned gossip copies stay legacy peers. Eviction removes a binding.

Requests stay unsigned, so `iam` is still a claim.
