# Peer judgements

The service judges a peer by its answer to a cross-check. A cross-check names
the documents the service wants a peer to confirm for one word. A peer that
lists only those documents honors the cross-check. A peer that lists documents
the cross-check did not name ignores it.

The service does not ask a peer that ignores the cross-check to cross-check
again until the software version the peer claims changes. When the answers of
a peer claim no version, the service asks the peer again after the retrial
interval. `YACYDHTSEARCH_PEER_RETRIAL_INTERVAL` in `configuration.md` sets
that interval.

The service holds the judgements in memory. An instance that starts again
holds no judgement and judges each peer again.

## Metrics

Both counters carry the label `question`, with the value
`lists only the cross-checked documents`.

| Metric | Label | Value | Meaning |
|---|---|---|---|
| `yacydhtsearch_word_joined_spread_peer_standings_total` | `standing` | `honoring` | the recorded judgement lets the service ask the peer |
| | | `ignoring` | the recorded judgement stops the service from asking the peer |
| | | `never judged` | no judgement is recorded, the service asks the peer to judge it |
| | | `version changed` | the peer claims a version other than the judged one, the service asks the peer again |
| | | `interval passed` | the retrial interval passed for a peer that claims no version, the service asks the peer again |
| `yacydhtsearch_word_joined_spread_peer_judgements_total` | `judged` | `honored` | the answer listed only named documents |
| | | `ignored` | the answer listed a document the cross-check did not name |
| | | `no evidence` | the answer listed no document, the recorded judgement stays |

A standing is counted each time the service decides if it asks a peer. A
judgement is counted for each cross-check the service put to a peer.
