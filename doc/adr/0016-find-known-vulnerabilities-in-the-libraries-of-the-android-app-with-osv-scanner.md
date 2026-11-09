# 16. Find known vulnerabilities in the libraries of the Android app with OSV-Scanner

Date: 2026-11-09

## Status

Accepted

## Context

The release APK carries Java and Kotlin libraries from Maven repositories. govulncheck checks
only Go code (ADR 0015). MASVS-CODE-3 asks that the app uses no component with a known
vulnerability (ADR 0014).

## Decision

OSV-Scanner 2.7.0, pinned in `tools/tools.lock`, checks the libraries of the release APK against
the OSV database. Gradle writes these libraries as a Gradle lockfile, so the check does not
include the build tools. The weekly CI workflow of ADR 0015 runs it.

## Consequences

- A new vulnerability does not stop an unrelated change, because `make android-app` does not run
  the check.
- The check sends the names and versions of the libraries to `api.osv.dev`.
