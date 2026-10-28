# 4. Sign the address fields of a seed

Date: 2026-10-28

## Status

Accepted

## Context

Seeds travel through Java peers that do not check them. Anyone can copy a peer's `Identity` into a
seed with another address.

## Decision

The seed column `AddressSignature` holds `<issued at>:<signature>`. The signature is Ed25519 by
the operational key over:

```text
"yacy-address-signature/1" ‖ length-prefixed(network name, peer hash,
    sorted normalized hosts, port, secure port or none, issued at)
```

The address fields stay in the seed. The column holds no address.

## Consequences

Stock Java YaCy 1.942 keeps the column and keeps hosts as the peer advertised them.

Java removes a host that it cannot reach from a seed and gossips that copy. The copy fails the
signature, so a verifier binds a peer from the seed that the peer sends itself.

Each seed grows by about 100 characters.
