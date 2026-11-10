# 18. Sign and check page attestations with the Hardware-Attested Statements verifier

Date: 2026-11-10

## Status

Accepted

## Context

pagescrape and pagerwi sign Hardware-Attested Statements, and the attestation check verifies
them (ADR 0016). A verification checks the Android key attestation chain, an RFC 3161
time-stamp over a Merkle tree, a drand round, and the policy of the format.

## Decision

The Go module `github.com/nikitakarpei/hardware-attested-statements/verifier` 0.2.0 gives the
bundle and statement types, the signed-statement ID, and the verification with its Android
profile. It is the reference verifier of the format.

## Consequences

- The services and the attestation check carry the verifier and the libraries that it uses:
  `github.com/digitorus/pkcs7` and `github.com/digitorus/timestamp` for the time-stamp,
  `github.com/transparency-dev/merkle` for the tree, and `github.com/cloudflare/circl` for the
  drand signature.
- The verifier needs Go 1.27.3 or later.
- A change of the format comes with a new version of the verifier.
