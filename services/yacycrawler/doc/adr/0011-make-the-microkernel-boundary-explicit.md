# 11. Make the microkernel boundary explicit

Date: 2026-07-23

## Status

Accepted

## Context

`internal/` held 21 flat packages with no namespace distinction between the crawl core and its
extension points. A single `crawlcapability` package held interfaces from roughly a dozen
unrelated seams, so "implements an extension point" and "is the crawl engine" looked identical at
a glance; the boundary lived only in `.go-arch-lint.yml` and the import graph, never in a
directory.

## Decision

`internal/crawl/` holds the kernel packages. Each declares its own extension-point interfaces and
crossing value types locally, next to the code that calls them — never in a standalone
contract-root package. `internal/crawl/` itself is a pure namespace, not a package.

Every seam a kernel package opens gets its own namespace directory directly under `internal/`,
named for both what its plugins are and of what, holding only plugin subdirectories. A plugin
imports the kernel package whose interface it satisfies; the kernel never imports a plugin. Where
several kernel packages need different event subsets from one plugin, each declares its own
minimal interface rather than sharing one contract. Where plugins duplicated transport plumbing,
the pure domain transform and the transport concern are split, with a single decorator adding
transport to any pure plugin instead of every plugin repeating it.

`.go-arch-lint.yml` declares one component per kernel package and per plugin, each plugin
restricted to the exact kernel package(s) whose interface it implements — never another plugin,
never another seam's kernel package.

## Consequences

A new format, media type, feed, policy, receiver, resolver, or observer is a new plugin under its
seam's namespace, registered in the composition root, touching no kernel file; the kernel's
membership test is lint-enforced rather than only conventional. Plugins in unrelated seams may
share a name; a collision is resolved with an import alias at the composition root, never by
renaming a plugin to avoid it. A seam gets a namespace even with a single plugin today, since seam
membership doesn't depend on how many plugins currently exist.
