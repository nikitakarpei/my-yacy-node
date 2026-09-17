# Cross-check with metadata

Status: proposal.

A cross-check asks a peer only for its listing: `urls=` and `abstracts=`, no
`query=`. The metadata of a joined document that no peer returned as a result
then costs a third round to `/yacy/urls.xml`.

A peer that honors `urls=` answers `query=<word>&urls=<documents>&count=<amount>`
with `abstracts=<word>` by the listing and one result row per named document it
holds, with its metadata and its posting. `yacynode` serves this today. This
proposal asks that form from the peers that honor it, so that the metadata
round covers only what no cross-check carried. It needs `cross-check-honoring.md`.

## Units

- `peerasks`, changed. A second ask kind beside `CrossCheckedDocumentsAsk`:
  `CrossCheckedDocumentsWithMetadataAsk`, whose answer carries the listing and
  the matched documents, each with its metadata and the count of the word.
- `peercallwire`, changed. One more single-ask call that puts the form above.
- `wordjoined`, changed. Puts the ask with metadata to the peers that honor
  `urls=`, and the listing-only ask to the peers on trial. The metadata round
  asks only for joined documents no answer carried metadata for.
- `yacynode/documentmatch`, changed. With required documents, reads the posting
  of each named document instead of scanning the word in impact order, so one
  ask costs the peer the size of the ask, not the size of the word.

## Boundaries

- Spread to honoring: which form each peer gets.
- Spread to wire: the ask with its documents; the answer with the listing and
  the matched documents.
- Spread to the answered query: a joined document that a cross-check matched
  carries the count of the cross-checked word before any page is read. Today it
  carries only the count of the leading word, or none.

## Rule of one cross-check

| Record of the peer | The ask |
|---|---|
| Honors `urls=` | with metadata; `count` is the amount of documents in the ask |
| On trial | listing only, as today |
| Ignores `urls=` | not put |

The listing decides which documents the peer holds, as today. The result rows
supply metadata and counts, and never add a document. The cross-checked
documents ceiling bounds the answer: one row is a few hundred bytes, so one
thousand rows stay far under the response byte ceiling. A YaCy peer never gets
this form: without `urls=` it would answer `count` rows of a plain search.

## Configuration

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_CROSS_CHECK_ASKS_FOR_METADATA` | `false` at landing, `true` after the rollout | Whether a cross-check of a peer that honors `urls=` also asks for the metadata of the documents it holds. `false` asks every peer for the listing only. |

## Metrics

New, published from `wordjoined` at zero from startup:

| Metric | Labels | Answers |
|---|---|---|
| `yacydhtsearch_word_joined_spread_joined_documents_with_metadata_from_a_cross_check_ratio` | none | the share of joined documents the cross-check saved from the metadata round |
| `yacydhtsearch_word_joined_spread_url_metadata_round_skipped_ratio` | none | how often the metadata round has nothing left to ask |

Read beside them
`yacydhtsearch_word_joined_spread_looked_up_documents_without_metadata_ratio`,
`yacydhtsearch_word_joined_spread_duration_seconds`, and
`yacydhtsearch_peer_call_duration_seconds` for the cost of the larger answers.

## Rollout

Land `yacynode/documentmatch` first, so an ask costs a node the size of the ask.
Land the service with the flag off, run for a period, then turn the flag on for
a period of the same length. Compare the spread duration and the two ratios
above. With the flag on, a metadata round that still runs names the peers that
ignore `urls=`, which is the share `cross-check-honoring.md` reports.
