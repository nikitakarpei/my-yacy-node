# 25. Prove postings under a postings root

Date: 2026-10-30

## Status

Accepted

## Context

A node knows a page version only by its fingerprint: the modified day and some small counts.
Any crawl can make a matching fingerprint. Thus a posting joins a stored version only from the
origin that sent the URL metadata. A node drops a copy of the same version from another holder,
and that copy is lost.

## Decision

A node that crawls a page puts the postings of the page version into a Merkle tree. The postings
root of the tree goes into the URL metadata. A node sends each posting to Go peers with a
postings proof that the posting is under that root. A version with a postings root takes proven
postings from all origins. It replaces a version of the same day without a root. The first root
of a day stays until a newer day arrives.

## Consequences

No node that passes on postings can add, change or move a posting in a version with a root.
Copies from other holders are not lost. A version without a root gets a root when a crawl with a
root arrives. The root does not show who crawled the page, or that the page held the words.

- Anyone can make a new version. The first copy of a URL costs no swap, and the swap limit
  applies to each address block.
- A Java peer removes the root. The node then drops the postings that Java peers send for that
  URL until a newer version arrives.
- Postings that wait for a version with a root can fill the escrow during an upgrade. The hold
  period sets a limit (ADR 0018).
- No standard defines the tree. The leaf encoding and the tree do not change without a new
  format version, which the root and the proof carry.
- During the upgrade, older Go nodes pass on roots but send no proofs. Receivers that check
  proofs drop those postings until the senders upgrade.
