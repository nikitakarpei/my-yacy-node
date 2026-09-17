# Cross-check by filter

Status: proposal.

A cross-check names its documents in `urls=`, twelve bytes each, and the
cross-checked documents ceiling bounds one ask at one thousand documents. The
documents of the leading word past the ceiling are never cross-checked.

A filter of the same documents costs about ten bits per document at one false
positive in a hundred, so one ask can name ten times the documents in the same
bytes. A peer tests each posting of the word against the filter and lists what
passes. A false positive lists a document the leading word does not hold, and
the join drops it, because a joined document must be listed for every word.

This proposal adds that form. It needs `cross-check-honoring.md`, which records
the filter form on its own: a node that honors `urls=` may not know the filter.

## Units

- `documentfilter`, new library. A filter built from documents at a false
  positive rate, that says whether it may hold a document. Its first kind is a
  Bloom filter under `documentfilters/bloom`. Its wire form names the kind.
- `yacyproto`, changed. Request field `urlsfilter`, the wire form of one filter.
  A YaCy peer ignores it and answers as it does to `urls=` today.
- `yacynode/postingfilter`, changed. Accepts a posting when the filter may hold
  its document. `indexabstract` scans and filters already, and does not change.
- `peerasks`, changed. A second ask kind beside `CrossCheckedDocumentsAsk`:
  `CrossCheckedDocumentsByFilterAsk`, with the filter in place of the documents.
- `peercallwire`, changed. One more single-ask call that puts the filter form.
- `wordjoined`, changed. Puts the filter form when the candidates pass the
  ceiling and the peer honors it.

## Boundaries

- Spread to honoring: which peers honor the filter form, or are on trial for it.
- Spread to the filter: the candidate documents and the rate in, one filter out.
- Spread to wire: the ask with its filter; the answer with the listing.
- Node endpoint to the posting filter: the filter from the request, beside the
  documents of `urls=`. A request that carries both admits what both admit.

## Rule of one cross-check

| Candidates of the leading word | Peer record for the filter form | The ask |
|---|---|---|
| Within the ceiling | any | named documents, as today |
| Past the ceiling | honors the form | one filter of all candidates up to the filter ceiling |
| Past the ceiling | on trial | one filter; listed documents outside the candidates past the rate judge the peer as ignoring |
| Past the ceiling | ignores the form | named documents up to the ceiling, as today |

Every peer of one word gets the same filter, where the named form deals the
candidates across the peers. A node still cuts a listing at one thousand
documents, so a peer that holds more of the candidates lists one thousand.

## Configuration

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_CROSS_CHECK_FILTER_DOCUMENTS_CEILING` | `10000` | Most documents one cross-check filter holds. |
| `YACYDHTSEARCH_CROSS_CHECK_FILTER_FALSE_POSITIVE_RATE` | `0.01` | Share of the documents outside a cross-check filter that it admits anyway. A lower rate makes the filter larger. |

## Metrics

New, published from `wordjoined` at zero from startup:

| Metric | Labels | Answers |
|---|---|---|
| `yacydhtsearch_cross_check_asks_total` | `form`: named documents, filter | how often the filter form is used |
| `yacydhtsearch_cross_check_filter_bytes` | none | what one filter costs on the wire |
| `yacydhtsearch_cross_check_filter_false_positives_ratio` | none | the share of a filter answer the join drops, against the configured rate |

Read beside them the past-the-ceiling ratio of the leading query word, which
the filter form should drive down, and the found-only-by-cross-checking ratio,
both of the word joined spread.

## Rollout

Land `documentfilter` and `yacynode` first, so nodes honor the form before any
service asks it. Then land the service; the first filter ask to each node is
its trial. Compare the two ratios above with the period before, and the peer
call duration for the cost of a scan of the whole word on each peer.
