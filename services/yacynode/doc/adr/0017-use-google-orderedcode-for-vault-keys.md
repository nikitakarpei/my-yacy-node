# 17. Use google/orderedcode for vault keys

Date: 2026-08-09

## Status

Accepted

## Context

A vault collection key is a sequence of domain values, such as a word hash and a
URL hash. The storage engine sorts entries by the key bytes and scans a byte
range. Two node behaviours depend on that sort: an escrow expiry sweep reads
every posting held before an instant, and a staleness sweep reads URLs from the
stalest rank. Both need the byte order of a key to equal the domain order of its
values.

A range scan also needs a partial key. A scan over one word reads every entry
whose key starts with the encoding of that word. This holds only if the encoding
of the first value is a byte prefix of the encoding of the whole sequence.

The former format concatenated fixed-width fields. It gave no escape for a
separator, so a value that contains the separator byte shifts the field
boundaries. It also rendered an instant as 20 decimal digits.

## Decision

Encode every vault key with `github.com/google/orderedcode`. The library
guarantees that byte-lexicographic order equals item-wise order, and that the
encoding of the first `i` items is a byte prefix of the encoding of the first
`j` items when `i < j`.

The package `internal/vaultkey` owns the type `Key` and the layout types
`SingleKey`, `PairKey` and `TripleKey`. One layout declaration drives encode,
decode and prefix, so a caller passes typed values and never handles bytes.

`Time` occupies two ordered positions, a signed 64-bit second count from the
Unix epoch and the nanosecond within that second. This keeps nanosecond
precision across the full `time.Time` range, including the zero instant. A
decoded instant is in UTC and carries no monotonic clock reading. It is equal to
the encoded instant, but it is not identical to it.

## Considered alternatives

An in-house escape and terminator encoding was rejected. The CockroachDB scheme
escapes `0x00` as `0x00 0xff` and terminates a value with `0x00 0x01`. This is
small but exact, and a single error in it corrupts a key range silently.

Fixed-width concatenation was kept until now and is rejected. It cannot hold a
variable-length value, it needs a length constant per field at every call site,
and it makes an instant 20 bytes wide.

## Consequences

`google/orderedcode` becomes a runtime dependency of the node. It has no
dependency of its own beyond the standard library.

The on-disk key format changes. The node does not migrate its bolt file. An
operator deletes the file, and the node rebuilds its index from crawl results
and from peer postings.
