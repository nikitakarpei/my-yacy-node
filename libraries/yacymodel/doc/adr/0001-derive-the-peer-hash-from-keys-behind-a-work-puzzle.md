# 1. Derive the peer hash from keys behind a work puzzle

Date: 2026-10-28

## Status

Accepted

## Context

YaCy lets a peer choose its hash. Nothing proves that a peer owns its hash, so any seed can put
any hash at any address. The hash is 72 bits wide, and stock Java peers use it for DHT positions,
so its width cannot change.

## Decision

A peer key has an operational key (Ed25519) and a recovery key of a hash-based signature scheme.

```text
recovery commitment = SHA-256(recovery public key)
key digest          = SHA-256("yacy-peer-id/1" ‖ Ed25519 public key ‖ recovery commitment)
H                   = SHA-256(key digest ‖ nonce)          nonce: 8 bytes, big-endian
peer hash           = H[0..8]   (72 bits)
fingerprint         = H[0..27]  (224 bits)
```

- The hash commits to the recovery key now. Shor's algorithm breaks Ed25519, and a later change
  of the hash would move every peer on the DHT ring and empty its data. With the commitment, the
  recovery key can later authorize a new operational key under the same hash.
- An identity is valid only when `H` ends with at least 28 zero bits. Without the puzzle, a
  second key for an existing hash costs 2^72 operations, which is in reach when an attacker can
  pick from many peers. The puzzle multiplies each try by 2^28.
- The zero bits are outside the hash bits. Zero bits inside the hash would cluster peers on the
  DHT ring and make collisions cheaper.
- 28 bits cost a new peer about 17 s on a 6-core ARM board. More bits do not protect existing
  hashes in a way that matters, and they slow every new peer.
- The minimum is a constant. It is not configurable and is never lowered.
- The key digest is computed once, so each try of the puzzle hashes one SHA-256 block.

The seed column `Identity` holds `<Ed25519 public key>:<recovery commitment>:<nonce>` in YaCy
base64. Each part has an exact length and must encode back to the same text, so one identity has
one text. This format is frozen.

A higher minimum later cannot apply to `H`: a new nonce that keeps the hash is a second preimage,
and a new key moves the peer on the DHT ring. So a higher cost for new identities comes from a
separate proof in a new seed column: a nonce `w` where `SHA-256("yacy-extra-work/1" ‖ H ‖ w)`
ends with a minimum of zero bits.

## Consequences

To take the hash of an existing peer costs about 2^100 SHA-256 operations, or about 2^52 with
Grover's algorithm. The 72-bit hash sets this limit, and no later change can raise it.

Two keys with one hash cost about 2^64. A verifier binds the full identity, not the hash, so such a
pair gives no advantage.

A new identity costs milliseconds on a GPU. The puzzle does not limit the number of identities
that one party makes.

The recovery key's scheme is fixed with the commitment. The recovery key is kept with the
operational key, so one step runs a peer; loss of both loses the identity, and theft of both gives
it to the thief.

Tests cannot use readable fixed hashes.
