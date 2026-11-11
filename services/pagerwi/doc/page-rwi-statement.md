# Page RWI statement

The page RWI statement is a predicate type of the Hardware-Attested Statements format. pagerwi
signs one for each RWI of a page it can attest, when it has a statement signer. The bundle
travels inside the RWI, so the subject comes from the URL metadata row, not from the bytes of
the RWI.

Predicate type:
`https://github.com/nikitakarpei/my-yacy-node/blob/main/services/pagerwi/doc/page-rwi-statement.md`

## Meaning

The signer fetched the page at the page URL over TLS, after the drand round in the statement. It
made the URL metadata row and the postings of the page from the response.

The page RWI root is a digest of the URL metadata row, except the day that the page was loaded.
The row holds the postings root, so the root covers the postings too. A verifier computes the
root from the row that it has.

## Statement

```json
{ "_type": "https://in-toto.io/Statement/v1",
  "subject": [{ "name": "pageRWIRoot", "digest": { "sha256": "<the page RWI root>" } }],
  "predicateType": "<the predicate type>",
  "predicate": {
    "pageURL": "https://example.org/a", "landedURL": "https://example.org/b",
    "startedAfter": { "drand": { "chain": "…", "round": 1, "signature": "…" } } } }
```

- `pageURL` is the page URL of the RWI.
- `landedURL` is the URL that the fetch landed at. It is absent when that is the page URL.
- `startedAfter` is the drand round of the page offer.
- The bundle has an RFC 3161 time-stamp.
- The statement lists no inputs. pagerwi signs only the page offers that its own device key
  attested.

## When pagerwi signs

pagerwi signs a statement only when all of these are true:

- the same device key signed the [page offer statement](../../pagescrape/doc/page-offer-statement.md),
  and the statement covers the page offer as received;
- the fetch started at the page URL, and the page URL is https on the default port. The fetch
  can follow redirects;
- the landed URL has 2048 characters or fewer;
- the drand round of the page offer is of the chain quicknet, and the fetch time of the page offer
  is the time of that round;
- the page has at least one word that is not a stopword.

Other RWIs have no attestation.
