# Scored abstracts

Status: proposal.

A listing names documents and nothing else. A YaCy peer cuts it at one thousand
documents in URL-hash order, so for a common word it is a sample with no order.
`yacynode` lists in impact order, and the impact stays in the node. The spread
takes the most listed documents first at each ceiling.

This proposal lets a peer send the score of each listed document when asked, so
that the spread deals the cross-check and the metadata round to the documents
that can reach the top. The score of a document for a word is the impact of its
posting, from the hits of the word and whether the word is in the title. Two
nodes give one posting the same score, so scores from different peers add.

## Units

- `yacyproto`, changed. Request field `abstractscores`, response key
  `indexabstractscore.<word>`: one integer per document of `indexabstract.<word>`,
  in the same order. A YaCy peer ignores both. A YaCy client ignores the key.
- `yacynode/indexabstract`, changed. With the request field, answers the impact
  of each listed document beside the listing.
- `peerasks`, changed. `AnsweredMatchedAndHeldDocumentsAsk` carries the score of
  each listed document, when the peer sent them.
- `wordjoined`, changed. Orders the documents it deals across the cross-check and
  the metadata round by their bounded score, and the rest as today.

Relevance stays as it is. The score orders only what the spread asks for.

## Boundaries

- Spread to wire: every matched-and-held ask sets `abstractscores`. The answer
  carries scores or not.
- Spread to peer choice and the join: nothing changes. The join needs no score.
- Spread to the observer: how many listings carried scores, and how many ranked
  items the score order let through a ceiling.

## Rule of one listing and one joined document

A listing with scores is in falling score order. When the peer holds more than
it listed, the lowest listed score bounds every document it did not list. The
bounded score of a joined document is the sum over the query words of its score
where a peer listed it with one, and of the bound of the peer where not.

| Joined document | Its place at a ceiling |
|---|---|
| A score for every query word | first, by the sum of its scores |
| A score for some query words | next, by the sum of what is known |
| No score for any query word | last, most listed first, as today |

The order applies to the cross-checked documents ceiling and the metadata
documents ceiling. Which pages are read comes from the ordering, as today, and
a score for it comes later.

## Configuration

None. The request field costs nothing on a peer that ignores it.

## Metrics

New, published from `wordjoined` at zero from startup:

| Metric | Labels | Answers |
|---|---|---|
| `yacydhtsearch_word_joined_spread_scored_listings_ratio` | none | how much of the network sends scores |
| `yacydhtsearch_word_joined_spread_joined_documents_ordered_by_score_ratio` | `scored`: every word, some words | how many joined documents the score can place |
| `yacydhtsearch_word_joined_spread_ranked_items_let_through_a_ceiling_by_score_ratio` | none | the share of ranked items that the listings order would have dropped at a ceiling, the quality signal |

Read beside them
`yacydhtsearch_word_joined_spread_joined_documents_dropped_before_metadata_lookup_ratio`
and
`yacydhtsearch_word_joined_spread_leading_query_word_documents_past_the_cross_checked_documents_ceiling_ratio`.

## Rollout

Land `yacynode` first. Until nodes hold a share of each word, every listing is
unscored and the spread orders as today. Then land the service, and read the
ranked-items ratio as the scored-listings ratio grows. A ratio near zero says
the listings order already finds what the ranking wants, and the score is worth
nothing more than the ordering of the pages read.
