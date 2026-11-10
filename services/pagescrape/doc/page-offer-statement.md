# Page offer statement

The page offer statement is a predicate type of the Hardware-Attested Statements format. With a
statement signer, pagescrape signs a statement for each page offer it can attest.

Predicate type:
`https://github.com/nikitakarpei/my-yacy-node/blob/main/services/pagescrape/doc/page-offer-statement.md`

## Meaning

The page offer with this digest is the fetch of `fetchURL` for the page URL of the offer. The
origin served the offered body, and the drand round in `startedAfter` came before the fetch.

## Statement

```json
{ "_type": "https://in-toto.io/Statement/v1",
  "subject": [{ "name": "pageOffer", "digest": { "sha256": "<SHA-256 of the offer message body>" } }],
  "predicateType": "<the predicate type>",
  "predicate": {
    "fetchURL": "https://example.org/a",
    "startedAfter": { "drand": { "chain": "…", "round": 1, "signature": "…" } } } }
```

- The digest covers the body of the page offer message exactly as pagescrape sends it.
- The statement lists no inputs. pagescrape signs only when its own key signed the response
  attestation of each hop.
- `startedAfter` is the drand round of the response attestation of the last hop.

## Where the bundle goes

The bundle travels in the header `Page-Offer-Attestation` of the page offer message. The body of
the message is the page offer, without the bundle. A corpus that reads only the body sees no
change.

## When pagescrape signs

pagescrape signs a statement only when all of these are true:

- each hop of the fetch has a response attestation, and the key of the signer signed it;
- each response attestation names the https form of the URL of its hop;
- the response attestation of the last hop names the offered body and a round of the drand
  chain quicknet.

The fetch time of the offer is then the time of that round. Any other page goes without an
attestation, with the time of the service clock as its fetch time. The service writes a log
line with the reason.

When the signer does not sign, the scrape request stays pending.
