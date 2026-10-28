# 1. Sign recovery keys with SLH-DSA from cloudflare/circl

Date: 2026-10-28

## Status

Accepted

## Context

The peer hash commits to a hash-based recovery key (ADR 0003). The standard library has
ML-DSA but no SLH-DSA.

## Decision

We use `github.com/cloudflare/circl/sign/slhdsa` with the parameter set SLH-DSA-SHA2-128s,
pinned in `go.mod`. `yacymodel` uses it to generate recovery keys and, in a later phase, to sign
and verify delegation certificates.

## Consequences

Each module that imports `yacymodel` gets circl in its module graph. SLH-DSA is FIPS 205, so a
move to another implementation keeps existing keys valid.
