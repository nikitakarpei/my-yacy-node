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

Each step signs a HAS statement about its output:

| Step | Statement | Subject |
| --- | --- | --- |
| Capture proxy | `http-response` | the body of the response |
| pagescrape | `page-offer` | the page offer message as sent |
| pagerwi | `page-rwi` | the page RWI root |

The page RWI root is SHA-256 over a domain byte, the digest of the URL metadata row, and the
postings root. The digest covers every field of the row except the postings root and the
`Loaded` day. The `page-rwi` statement also holds the page URL, the landed URL when it is not
the page URL, and the drand round of the capture. One RFC 3161 time-stamp covers it. Its bundle
is the page attestation.

A step signs only when its input data matches the subject of the input statement. For the
signer of the input, the step does one of two things:

- It vouches and lists no input. Today a step vouches only for its own key.
- It lists the input statement and nests the input bundle.

pagerwi attests a page only when the fetch URL is the page URL, and the page URL is https on
the default port. Every other page gets an RWI without attestation.

pagerwi takes the `Loaded` day from the fetch time that the page offer carries. The fetch time
is the time of the drand round of the capture, or the clock of pagescrape without a capture.

pagescrape sends the bundle of the `page-offer` statement in a header of the page offer
message, beside the offer, so the digest covers the body exactly as sent.

The program that embeds pagescrape and pagerwi gives them a statement signer. Without one,
they publish no attestation. The capture proxy of the app does not time-stamp captures.

The attestation check is a library. It takes a URL metadata row, a page attestation and a
policy, and gives "attested" with the fetch time, or "not attested" with the reason. A policy
lists the app versions it accepts. Each listed version can have an end time: the policy then
accepts the version only in statements time-stamped before that time.

The page attestation goes on the RWI stream. Nodes do not keep it or send it to peers yet.

## Consequences

- No peer verifies an attestation until nodes carry it. That design is open.
- The release key of the app can sign any page. ADR 0017 makes the official app.
- An attestation shows that the origin served the page to the device over TLS. It does not
  show that the page is true, the same for each visitor, or not spam.
- Pages without postings, pages through renderproxy, spamproxy or the web archive, pages with
  an http page URL, and offers signed by an earlier key of the app get no attestation.
- A page attestation is about 5 KB.
- The attestation needs the time-stamp authority, drand, and the revocation list of Google.
