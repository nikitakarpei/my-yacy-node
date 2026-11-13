# 2. Sign each release with a new key

Date: 2026-11-13

## Status

Accepted. Supersedes the release key of [ADR 1](0001-release-the-companion-app-from-ci-as-an-immutable-release.md).

## Context

Android key attestation names an app by its package, its versionCode and its signing
certificate, not by a hash of the APK. Whoever holds a key that peers trust can sign a build
that is not published and make valid page attestations with it.

## Decision

The release workflow has a build job and a signing job. The signing job runs no Gradle and no
make target. It makes a new key, signs the unsigned APK of the build job, and deletes the key.

The release record holds the SHA-256 digest of the signing certificate and the versionCode.
The signing job attests the build provenance of the unsigned APK, the signed APK and the
release record. The release holds these three files and the Sigstore bundle of the attestation.

## Consequences

- No person holds a key that peers trust. Peers can trust a release from its provenance: the
  workflow of this repository at a release tag.
- A workflow run that copies its key out lets the key holder sign builds until peers stop
  trusting that release. The run is public.
- Android accepts no update signed with another key. Each update of the app is uninstall,
  install, and pair again.
