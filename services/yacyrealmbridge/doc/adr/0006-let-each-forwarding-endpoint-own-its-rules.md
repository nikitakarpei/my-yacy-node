# 6. Let each forwarding endpoint own its rules

Date: 2026-09-14

## Status

Accepted

## Context

The first draft named a boundary rule engine and a catalogue of refusal answers. Each
peer-protocol path refuses in its own way: a transfer answers not granted with a pause, a hello
answers a virgin peer type. The rules are few and read from one configuration.

## Decision

Each peer-protocol path is one endpoint. It reads the operator's configuration for its crossing
direction, admits or refuses, answers a refusal in the vocabulary of its path, and forwards
otherwise. There is no rule engine and no shared refusal catalogue.

## Consequences

Adding a path adds one endpoint. A rule that spans paths is repeated in each endpoint, and that
repetition is the price. Configuration stays a list of admitted and refused paths per crossing
direction.
