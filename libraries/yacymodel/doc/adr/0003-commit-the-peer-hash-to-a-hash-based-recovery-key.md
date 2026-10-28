# 3. Commit the peer hash to a hash-based recovery key

Date: 2026-10-28

## Status

Accepted

## Context

Shor's algorithm breaks Ed25519. The peer hash derives from the key, so a later change of key
algorithm would change every peer hash, move every peer on the DHT ring, and make each node start
over with an empty data directory. Only a commitment made now avoids that.

## Decision

Each peer key holds an SLH-DSA-SHA2-128s recovery key, and the peer hash commits to its public key
(ADR 0002). Nothing signs with the recovery key yet.

In a later phase, the recovery key signs a delegation certificate:
`("yacy-peer-delegation/1", fingerprint, algorithm, K, seq)` for an operational key `K`.

- `K` is outside the hash, so its algorithm can change without a change of hash.
- The recovery key is hash-based, so the chain depends only on hash functions.
- Certificates have no expiry. Expiry would need regular signatures by the recovery key. A
  pinned `seq` stops a return to an old certificate.
- Certificates travel through a peer endpoint, never in seeds: with a certificate, a seed
  passes the 16000-character limit of Java peers.

Rules for verifiers:

- A verifier pins the highest `seq` for a peer and refuses a lower one.
- Two different certificates with one `seq`: the first stays and the verifier logs an ERROR.
- After a verifier pins a certificate for a peer, it refuses that peer's seeds without one.
- A move of the whole network to certificates only is a separate decision.

## Consequences

A move to a new key algorithm, and recovery after theft of the operational key, keep the peer
hash.

The SLH-DSA parameter set is fixed now.

The recovery key is kept with the operational key, so one step runs a peer. Loss of both loses
the identity, and theft of both gives it to the thief.
