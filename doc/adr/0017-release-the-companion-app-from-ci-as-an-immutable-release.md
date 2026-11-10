# 17. Release the companion app from CI as an immutable release

Date: 2026-11-10

## Status

Accepted

## Context

Android key attestation names an app by its package and its signing certificate. Peers can
trust the page attestations of an app only when they know which certificate signs the official
app. Before this decision, each operator signed the APK with an own key.

## Decision

There is one official app. A release workflow of this repository:

1. builds the release APK and checks that two builds give the same unsigned APK;
2. signs the APK with the release key, which the owner of the repository holds;
3. attests the build provenance of the signed and the unsigned APK with `actions/attest`;
4. publishes both APKs as a GitHub release, which the repository makes immutable.

The release key is a secret of a GitHub environment that only the owner can approve.

## Consequences

- The default policy of the attestation check (ADR 0016) names the certificate of the release
  key.
- Whoever holds the release key can sign any build, also one that is not published.
- Anyone can rebuild the unsigned APK from the tagged commit and compare it with the released
  one. The provenance, not a rebuild, shows that the signed APK holds the same build.
- A release cannot change after it is published, and its tag cannot be used again.
- Release immutability is a setting of the repository, which the owner turns on.
