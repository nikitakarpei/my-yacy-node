# 3. Hold rankings in memory with golang-lru

Date: 2026-09-06

## Status

Accepted

## Context

An operator who runs one instance must not need a message broker to get consistent pages.

## Decision

We hold rankings in this process with `github.com/hashicorp/golang-lru/v2`, whose expirable LRU
gives a capacity and a lifetime together. It carries no dependencies of its own.

## Consequences

The service holds rankings with no infrastructure. Rankings do not survive a restart and are
not shared between instances.
