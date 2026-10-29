# 3. Delegate operational keys with recovery-key certificates

Date: 2026-10-28

## Status

Proposed. Not built. The open options below are decided before it is built.

## Context

The peer hash commits to a recovery key (ADR 0001). A peer needs a way to replace its operational
key, for a new algorithm or after theft, without a change of hash.

## Decision

The recovery key signs a delegation certificate:
`("yacy-peer-delegation/1", fingerprint, algorithm, K, seq)` for an operational key `K`.

- `K` is outside the hash, so its algorithm can change without a change of hash.
- The recovery key is hash-based (ADR 0002), so the chain depends only on hash functions.
- Certificates travel through a peer endpoint, never in seeds: with a certificate, a seed
  passes the 16000-character limit of Java peers.

Rules for verifiers:

- A verifier pins the highest `seq` for a peer and refuses a lower one.
- Two different certificates with one `seq`: the first stays and the verifier logs an ERROR.
- After a verifier pins a certificate for a peer, it refuses that peer's seeds without one.

## Open options

- Expiry of a certificate. Expiry limits the life of an old certificate at a verifier that never
  saw a newer one. A peer whose renewal fails drops out, and validity depends on clocks.
- A move of the whole network to certificates only: when, and what starts it.

## Consequences

A move to a new key algorithm, and recovery after theft of the operational key, keep the peer
hash.

A certificate is about 8 KB, so a verifier fetches it once per peer and keeps it.
