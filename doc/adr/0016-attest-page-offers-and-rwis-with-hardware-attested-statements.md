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

The page RWI root is SHA-256 over a domain byte and the URL metadata row, without its `Loaded`
day. The row holds the postings root, so the root covers the postings too. The `Loaded` day is
left out for two reasons: a crawl of an unchanged page then gives the same root, and the
statement holds the fetch time itself. The `page-rwi` statement also holds the page URL, the
landed URL when it is not the page URL, and the drand round of the capture. One RFC 3161
time-stamp covers it. Its bundle is the page attestation.

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
policy in the HAS format, and gives "attested" with the fetch time, or "not attested" with the
reason. The policy names the app and the app versions it accepts.

The page attestation goes on the RWI stream. Nodes do not keep it or send it to peers yet.

## Consequences

- No peer verifies an attestation until nodes carry it. That design is open.
- Peers can trust the attestations only of an app whose signing certificate they know. Whoever
  holds that key can sign any page.
- The `Loaded` day of an attested row is not covered: a peer that passes the row on can change
  it. The fetch time comes from the check, not from the row.
- An attestation shows that the origin served the page to the device over TLS. It does not
  show that the page is true, the same for each visitor, or not spam.
- Pages without postings, pages through renderproxy, spamproxy or the web archive, pages with
  an http page URL, and offers signed by an earlier key of the app get no attestation.
- A page attestation is about 5 KB.
- The attestation needs the time-stamp authority, drand, and the revocation list of Google.
