# 4. Sign the address fields of a seed

Date: 2026-10-28

## Status

Accepted

## Context

Seeds travel through Java peers that do not check them. `Identity` (ADR 0002) is public, so
anyone can copy it into a seed with another address. A peer can also claim an address in its own
hello and answer the callback there.

## Decision

The seed column `AddressSignature` holds `<issued at>:<signature>`. The signature is Ed25519 by
the operational key over:

```text
"yacy-address-signature/1" ‖ length-prefixed(network name, peer hash,
    sorted normalized hosts, port, secure port or none, issued at)
```

- The signature covers the address fields that the seed already holds, so no address is stated
  twice. Java keeps these fields as the peer advertised them.
- Hosts are signed as a sorted set, because Java moves hosts between `IP` and `IP6`.
- Other seed fields are not signed, because Java rewrites some of them, such as `LastSeen` and
  `PeerType`.
- `issued at` lets a verifier refuse an older address. Java rewrites `LastSeen`, so the column
  holds its own time.
- Both sides know the network name, so it is signed but not carried.

## Consequences

Stock Java YaCy 1.942 keeps the column.

Java removes a host that it cannot reach from a seed and gossips that copy. The copy fails the
signature, so a verifier binds a peer from the seed that the peer sends itself.

Each seed grows by about 100 characters.
