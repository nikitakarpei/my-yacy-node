# 6. Keep the metric history in Pebble

Status: accepted

## Context

`yacypeer` keeps its metric history on disk for a retention period. The history must survive
a restart and delete old samples without a full rewrite.

## Decision

Keep the history in Pebble through `libraries/vault` and `pebblevault`, in the data
directory of the peer.

## Consequences

* The history uses the storage engine that the services already use.
* Pebble compacts the history in the background.
