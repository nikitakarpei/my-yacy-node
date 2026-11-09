# 15. Find known vulnerabilities in the Go modules with govulncheck

Date: 2026-11-09

## Status

Accepted

## Context

The services, the libraries and the Android app are Go modules. Their dependencies and the Go
standard library can have known vulnerabilities. MASVS-CODE-3 asks that the Android app uses no
component with a known vulnerability (ADR 0014).

## Decision

govulncheck 1.9.0 from `golang.org/x/vuln`, pinned as a Go tool directive in the Go module
`tools`, checks each Go module of the repository against the Go vulnerability database. It
reports only the vulnerable functions that the code of the module can call. A CI workflow runs it
each week.

## Consequences

- A new vulnerability does not stop an unrelated change, because `make verify` does not run the
  check.
- The end-to-end test modules are not checked, because they do not ship.
- The check reads the Go vulnerability database from `vuln.go.dev`.
