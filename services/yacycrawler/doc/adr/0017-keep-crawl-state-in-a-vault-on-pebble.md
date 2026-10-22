# 17. Keep crawl state in a vault on Pebble

Date: 2026-10-22

## Status

Accepted

## Context

The crawler keeps its crawls, the frontier and the state of each page visit. A
Redis-protocol server added a stack container and a network hop to every page visit, while
production runs one crawler instance. The repository already has a transactional store
abstraction, the vault, with an embedded Pebble engine.

## Decision

The crawler keeps its crawl state in a vault (`libraries/vault`) on the Pebble engine
(`libraries/vaultengines/pebblevault`, which pins `github.com/cockroachdb/pebble/v2`). The
crawler core opens each transaction and passes it to the stores. The stores code against the
vault contract only, so another engine can replace Pebble. JetStream keeps only streams.

## Consequences

Crawl state lives in a directory of one crawler instance, with bounded memory. Two instances
do not share state. A shared engine behind the vault is needed before instances can share a
frontier. Store tests run against the vault engines and need no external infrastructure.
