# 16. Attest page offers and RWIs with Hardware-Attested Statements

Date: 2026-11-10

## Status

Accepted

## Context

Peers of the YaCy network cannot fetch again each page that another peer indexed. The capture
proxy of the companion app signs each origin response with a hardware-attested key, in the
Hardware-Attested Statements (HAS) format. pagescrape and pagerwi do not sign their output, so
a peer cannot tell that an RWI comes from a real page.

## Decision

Each step of the pipeline signs a HAS statement about its output. The capture proxy signs the
response, pagescrape signs the page offer, and pagerwi signs the RWI of the page. The bundle of
the last statement is the page attestation.

A step signs only when its input data matches the subject of the input statement. For the
signer of the input, the step does one of two things:

- It vouches and lists no input. Today a step vouches only for its own key.
- It lists the input statement and nests the input bundle.

pagescrape and pagerwi sign only when they have a statement signer. Without one, they publish
no attestation.

The attestation check is a library. It takes a URL metadata row, a page attestation and a
policy in the HAS format, and gives "attested" with the fetch time, or "not attested" with the
reason.

The page attestation goes on the RWI stream. Nodes do not keep it or send it to peers yet.

## Consequences

- No peer verifies an attestation until nodes carry it. That design is open.
- Peers can trust the attestations only of an app whose signing certificate they know. Each
  release of the companion app has a new key, see
  [its ADR 2](../../distributions/yacypeercompanion/doc/adr/0002-sign-each-release-with-a-new-key.md).
- An attestation shows that the origin served the page to the device over TLS. It does not
  show that the page is true, the same for each visitor, or not spam.
- Pages through renderproxy, spamproxy or the web archive get no attestation.
- A page attestation is about 5 KB.
- The attestation needs the time-stamp authority, drand, and the revocation list of Google.
