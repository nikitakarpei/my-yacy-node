# Page RWI statement

The page RWI statement is a predicate type of the Hardware-Attested Statements format. A
program that embeds pagerwi gives it a statement signer. pagerwi then signs a statement for each
RWI it can attest, and puts the bundle, the page attestation, in the `Attestation` member of the
RWI.

Predicate type:
`https://github.com/nikitakarpei/my-yacy-node/blob/main/services/pagerwi/doc/page-rwi-statement.md`

## Meaning

The signer fetched the page at `pageURL` over TLS, after the drand round in `startedAfter`. It
derived the URL metadata row and the postings of the page RWI root from the response.

The page RWI root is SHA-256 over the byte 1 and every field of the URL metadata row, the postings
root included, except the day the page was loaded. A verifier computes the root from the row
that it has.

## Statement

```json
{ "_type": "https://in-toto.io/Statement/v1",
  "subject": [{ "name": "pageRWIRoot", "digest": { "sha256": "<the page RWI root>" } }],
  "predicateType": "<the predicate type>",
  "predicate": {
    "pageURL": "https://example.org/a", "landedURL": "https://example.org/b",
    "startedAfter": { "drand": { "chain": "…", "round": 1, "signature": "…" } } } }
```

- `landedURL` is absent when the fetch landed at the page URL.
- The statement lists no inputs: the key that signs it also signed the page offer statement.
- The bundle has an RFC 3161 time-stamp.

## When pagerwi signs

pagerwi signs a statement only when all of these are true:

- the key of the signer signed the [page offer statement](../../pagescrape/doc/page-offer-statement.md)
  of the offer, and that statement covers the offer message as received;
- the fetch URL is the page URL, and the page URL is https on the default port;
- the landed URL has 2048 characters or fewer;
- the drand round of the offer is of the chain quicknet, and the fetch time of the offer is its
  time;
- the page has postings.

Any other RWI goes without a page attestation. The service writes a log line with the reason.
When the signer does not sign, the offered page stays pending.
