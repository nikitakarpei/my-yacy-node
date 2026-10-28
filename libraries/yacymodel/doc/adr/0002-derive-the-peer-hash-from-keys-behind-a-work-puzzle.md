# 2. Derive the peer hash from keys behind a work puzzle

Date: 2026-10-28

## Status

Accepted

## Context

YaCy lets a peer choose its hash. Nothing proves that a peer owns its hash, so any seed can put
any hash at any address.

## Decision

```text
recovery commitment = SHA-256(recovery public key)
key digest          = SHA-256("yacy-peer-id/1" ‖ Ed25519 public key ‖ recovery commitment)
H                   = SHA-256(key digest ‖ nonce)          nonce: 8 bytes, big-endian
peer hash           = H[0..8]   (72 bits)
fingerprint         = H[0..27]  (224 bits)
```

An identity is valid only when `H` ends with at least 28 zero bits. The minimum is a constant. It
is not configurable and is never lowered.

The seed column `Identity` holds `<Ed25519 public key>:<recovery commitment>:<nonce>` in YaCy
base64. Each part has an exact length and must encode back to the same text. This format is
frozen.

To raise the cost of a new identity later, a separate proof is added in a new seed column: a nonce
`w` where `SHA-256("yacy-extra-work/1" ‖ H ‖ w)` ends with a minimum of zero bits. The hash
does not change.

## Consequences

To take the hash of an existing peer costs about 2^100 SHA-256 operations, or about 2^52 with
Grover's algorithm. YaCy's 72-bit hash sets this limit, and no later change can raise it.

Two keys with one hash cost about 2^64. A verifier binds the full identity, not the hash, so such a
pair gives no advantage.

A new identity costs about 2^28 operations: about 17 s on a 6-core ARM board, milliseconds on a
GPU. The puzzle does not limit the number of identities that one party makes.

Tests cannot use readable fixed hashes.
