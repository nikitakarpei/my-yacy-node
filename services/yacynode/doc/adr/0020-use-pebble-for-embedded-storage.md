# 20. Use Pebble for embedded storage

Date: 2026-09-05

## Status

Accepted

Supersedes the engine choice in [5. Use bbolt for embedded storage](0005-use-bbolt-for-embedded-storage.md).
Keeps the ownership rules of [10. Own the embedded database behind a storage kernel](0010-boltvault-storage-kernel.md).

## Context

The node must hold a 100GB corpus on a small Linux board with about 1GB of free memory and an
SSD. bbolt cannot reach that size. It writes the full free page list on every commit, so the
cost of one commit increases with the size of the database, and eviction makes that list large.
The mapped file also makes the memory the node uses impossible to budget and to observe.

The engine must scan keys in order, commit several collections in one transaction, and let the
operator set how much memory the storage takes. The build must stay pure Go, because the board
is cross-compiled and the quality gate has no C toolchain.

## Decision

Use `github.com/cockroachdb/pebble` as the embedded storage engine of the node.

The node opens `pebblevault` instead of `boltvault`. `boltvault` stays in the repository as an
implementation of the vault engine interface, but no service starts it.

The operator sets the memory for cached blocks and for buffered writes, the number of
compactions that run at the same time, and the number of open table files.

## Considered alternatives

Keep bbolt. Rejected because the commit cost is structural. The one option that removes the
free page list write makes the node read the full file at each start.

Badger. Rejected because its value log needs a garbage collection that the node must schedule
and observe. That is a second reclamation cycle beside the eviction the node already owns.

goleveldb. Rejected because it has no maintenance and no user that runs it at this size.

RocksDB. Rejected because it needs cgo. That ends the pure Go cross build and adds a C
toolchain to every build and to the quality gate.

SQLite. Rejected because it adds a schema and a migration policy for an access pattern that is
ordered key and value only. The pure Go port is a machine translation of the C source, which is
a larger surface to trust than Pebble. The alternative port needs cgo.

LMDB. Rejected because it maps the file like bbolt, so the memory stays unbudgetable, and its
Go binding needs cgo.

A private file format. Rejected because crash recovery and compaction are the difficult parts,
and they are what the node would have to write.

## Consequences

Commit cost no longer increases with the size of the database, and the operator can set the
memory the storage takes.

Compaction writes each record more than once and keeps space that deleted records used until it
runs. The quota therefore counts the stored keys and values, not the disk the engine uses. The
operator must give the disk headroom above the quota.

A node that runs bbolt cannot read its data with Pebble. The operator starts with an empty
index and crawls again.

The node depends on a larger set of packages than before.
