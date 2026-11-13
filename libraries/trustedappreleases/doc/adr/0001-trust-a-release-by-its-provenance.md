# 1. Trust a release by its provenance

Date: 2026-11-13

## Status

Accepted

## Context

Each release of an app has a new signing key
([companion ADR 2](../../../../distributions/yacypeercompanion/doc/adr/0002-sign-each-release-with-a-new-key.md)).
Peers must tell which signing certificates come from the release workflow. Anyone with write
access to the repository can upload release assets and attestations by hand.

## Decision

A release is trusted only from the verified provenance of its release record. The provenance
is a Sigstore bundle that GitHub Actions attests. Its certificate must name the repository and
the owner by numeric ID, the release workflow at the tag of the release, the commit of the tag,
a runner that GitHub hosts, and a public repository.

The Go module `github.com/sigstore/sigstore-go` 1.4.0 verifies the bundle against the Sigstore
trusted root: the certificate chain, a certificate timestamp, the transparency log entry, and
the subject of the statement.

## Consequences

- The Sigstore trusted root and GitHub are trust anchors.
- The library carries the dependencies of sigstore-go, which are many.
- A release that is not immutable is not trusted, also when its provenance verifies.
