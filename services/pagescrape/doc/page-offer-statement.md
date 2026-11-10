# Page offer statement

The page offer statement is a predicate type of the Hardware-Attested Statements format. A
program that embeds pagescrape gives it a statement signer. pagescrape then signs a statement
for each page it can attest, and puts the bundle in the `Attestation` member of the offer.

Predicate type:
`https://github.com/nikitakarpei/my-yacy-node/blob/main/services/pagescrape/doc/page-offer-statement.md`

## Meaning

The page offer of pagescrape is the fetch of `fetchURL` for `pageURL`. The fetch landed at
`landedURL`, and the origin served this body with these headers.

## Statement

```json
{ "_type": "https://in-toto.io/Statement/v1",
  "subject": [{ "name": "body", "digest": { "sha256": "<SHA-256 of the body>" } }],
  "predicateType": "<the predicate type>",
  "predicate": {
    "pageURL": "https://example.org/a", "fetchURL": "https://example.org/a",
    "landedURL": "https://example.org/b", "contentType": "text/html",
    "robotsTagValues": ["noarchive"], "pageModifiedAt": "2026-10-30T01:02:03Z",
    "inputVouching": "sameKey",
    "startedAfter": { "drand": { "chain": "…", "round": 1, "signature": "…" } } } }
```

- `robotsTagValues` and `pageModifiedAt` are absent when the offer has none.
- `inputVouching` is `sameKey`: the key that signs the statement also signed the capture of each
  hop. The statement lists no inputs.
- `startedAfter` is the drand round of the capture of the last hop.

## When pagescrape signs

pagescrape signs a statement only when all of these are true:

- each hop of the fetch has a capture, and the key of the signer signed it;
- each capture names the https form of the URL of its hop;
- the capture of the last hop names the offered body and a round of the drand chain quicknet.

The fetch time of the offer is then the time of that round. Any other page goes without an
attestation, with the time of the service clock as its fetch time. The service writes a log
line with the reason.

When the signer does not sign, the scrape request stays pending.
