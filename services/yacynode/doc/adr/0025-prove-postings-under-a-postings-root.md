# 25. Prove postings under a postings root

Date: 2026-10-30

## Status

Accepted

Amends ADR 0008.

## Context

A node knows a page version only by its fingerprint: the modified day and some small counts.
Any crawl can make a matching fingerprint. Thus a posting joins a stored version only from the
origin that sent the URL metadata. A node drops a copy of the same version from another holder,
and that copy is lost.

## Decision

A node that crawls a page puts the postings of the page version into a Merkle tree and puts the
postings root into the URL metadata. It sends each posting to Go peers with a postings proof under
that root. A version with a root takes proven postings from all origins and replaces any version
without a root. From a peer, only a version with a root and a newer day replaces it. To make
storage available, the node removes versions without a root first.

## Consequences

No node that passes on postings can add, change or move a posting in a version with a root.
Copies from other holders are not lost. A version without a root gets a root when a crawl with a
root arrives. A crawl of this node replaces any version. The root does not show who crawled the
page, or that the page held the words.

- Anyone can make a new version. The first copy of a URL costs no swap, and the swap limit
  applies to each address block.
- A Java peer removes the root. The node then drops the postings and the URL metadata that Java
  peers send for a URL whose version has a root. If only Java peers crawl the page again, the
  node keeps the older version with a root. Only a version with a root and a newer day replaces
  it.
- Anyone can send URL metadata with a root for new URLs at no swap cost. When storage is full,
  the node removes every URL without a root before those records. This includes new content
  that Java peers send.
- Postings that wait for a version with a root can fill the escrow during an upgrade. The hold
  period sets a limit (ADR 0018).
- No standard defines the tree. The leaf encoding and the tree do not change without a new
  format version, which the root and the proof carry.
- During the upgrade, older Go peers pass on roots but send no proofs. Receivers that check
  proofs drop those postings until the senders upgrade.
- A node keeps the postings proof of a URL and the postings of the URL in different stores.
  Only the write and purge paths keep the proof correct for the postings. If the proof is not
  correct, the node sends the postings of that URL with no proof or with a proof that does not
  lead to the root.
