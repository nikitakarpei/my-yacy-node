# 2. Sign recovery keys with SLH-DSA from cloudflare/circl

Date: 2026-10-28

## Status

Accepted

## Context

ADR 0001 gives each peer a recovery key of a hash-based signature scheme. That key must stay
secure when quantum computers break Ed25519, and when a new attack breaks lattice-based schemes
such as ML-DSA. The standard library has ML-DSA but no SLH-DSA.

## Decision

We use `github.com/cloudflare/circl/sign/slhdsa` with the parameter set SLH-DSA-SHA2-128s,
pinned in `go.mod`.

- SLH-DSA depends only on hash functions.
- SLH-DSA is stateless, so a peer restored from a backup can sign again safely.
- The "s" parameter sets have the smallest signatures (about 8 KB). Signing is slow, and a
  recovery key signs rarely.

## Consequences

Each module that imports `yacymodel` gets circl in its module graph. SLH-DSA is FIPS 205, so a
move to another implementation keeps existing keys valid.
