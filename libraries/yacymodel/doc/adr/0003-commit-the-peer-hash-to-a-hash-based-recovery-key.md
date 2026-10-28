# 3. Commit the peer hash to a hash-based recovery key

Date: 2026-10-28

## Status

Accepted

## Context

Shor's algorithm breaks Ed25519. The peer hash derives from the key, so a change of key algorithm
changes every peer hash.

## Decision

Each peer key holds an SLH-DSA-SHA2-128s recovery key, and the peer hash commits to its public key
(ADR 0002). Nothing signs with the recovery key yet.

In a later phase, the recovery key signs a delegation certificate:
`("yacy-peer-delegation/1", fingerprint, algorithm, K, seq)`, without expiry, for an operational
key `K` of any algorithm. Certificates travel through a peer endpoint and never in seeds.

- A verifier pins the highest `seq` for a peer and refuses a lower one.
- Two different certificates with one `seq`: the first stays and the verifier logs an ERROR.
- After a verifier pins a certificate for a peer, it refuses that peer's seeds without one.
- A move of the whole network to certificates only is a separate decision.

## Consequences

A move to a new key algorithm, and recovery after theft of the operational key, keep the peer
hash. The chain depends only on hash functions.

The SLH-DSA parameter set is fixed now. A delegation certificate is about 8 KB.

The recovery key is kept with the operational key. Loss of both loses the identity, and theft of
both gives it to the thief.
