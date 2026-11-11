# Page offer statement

The page offer statement is a predicate type of the Hardware-Attested Statements format.
pagescrape signs one for each page offer it can attest, when it has a statement signer.

Predicate type:
`https://github.com/nikitakarpei/my-yacy-node/blob/main/services/pagescrape/doc/page-offer-statement.md`

## Meaning

The page offer with this digest is a fetch of the fetch URL. The origin served the offered
content, and the fetch started after the drand round in the statement.

## Statement

```json
{ "_type": "https://in-toto.io/Statement/v1",
  "subject": [{ "name": "pageOffer", "digest": { "sha256": "<SHA-256 of the page offer>" } }],
  "predicateType": "<the predicate type>",
  "predicate": {
    "fetchURL": "https://example.org/a",
    "startedAfter": { "drand": { "chain": "…", "round": 1, "signature": "…" } } } }
```

- The digest covers the page offer exactly as pagescrape publishes it.
- `fetchURL` is the URL that the fetch started at.
- `startedAfter` is the drand round that the response attestation of the last hop names.
- The statement lists no inputs. pagescrape signs only the fetches that its own device key
  attested.

## When pagescrape signs

pagescrape signs a statement only when all of these are true:

- the same device key attested the response of each hop of the fetch;
- each response attestation names the URL of its hop;
- the response attestation of the last hop names the offered content and a round of the drand
  chain quicknet.

The fetch time of an attested page offer is the time of that round. Other page offers have no
attestation.
