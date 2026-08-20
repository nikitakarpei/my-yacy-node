# 19. Extract the vault as libraries

Date: 2026-08-20

## Status

Accepted

Supersedes the ownership and the naming of
[10. Own the embedded database behind a storage kernel](0010-boltvault-storage-kernel.md).

## Context

The storage kernel, the key encoding, the two engine drivers and the driver
conformance suite were packages inside the node. They hold no node vocabulary,
and the node carried their dependencies.

## Decision

The kernel and the key encoding are one library module, `vault`, because a caller
cannot use a collection without the key encoding.

Each driver is a library module of its own: `boltvault` and `memoryvault`. A
driver depends on the contract; the contract depends on no driver.

## Consequences

Another service can store data in a vault without a dependency on the node, and a
deployment reads only the driver it wires.

Each module passes the quality gate on its own.

A change to the `Engine` contract now crosses a module boundary. The conformance
suite reports a driver that does not follow.
