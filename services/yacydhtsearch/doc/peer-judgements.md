# Peer judgements

The service learns, for each peer and for each form of an ask, if the peer
honors that form. A peer tells nothing about itself. Its answer is the
evidence.

The only form today is the named documents of a cross-check ask. The service
asks a peer about a word and names the documents it wants. A peer that honors
the form lists only the named documents. A peer that ignores the form lists
every document it holds for the word. The service keeps the documents of both
answers.

An answer can carry the software version of the peer. The service records that
version with the judgement. A judgement holds until the peer claims a different
version. The service asks only the peers that honor the form and the peers it
must judge again.

## The standing of one peer in one form

| Recorded judgement | Version the peer claims now | Standing |
|---|---|---|
| None | any | never judged: the service asks the peer |
| Judged at version `v` | `v` | as judged: the service asks a peer that honors, and does not ask a peer that ignores |
| Judged at version `v` | a different version | version changed: the service asks the peer |
| Judged, and one of the two versions is empty, less than `YACYDHTSEARCH_PEER_RETRIAL_INTERVAL` ago | | as judged |
| Judged, and one of the two versions is empty, `YACYDHTSEARCH_PEER_RETRIAL_INTERVAL` ago or more | | interval passed: the service asks the peer |

A peer that gave no answer to the first ask of the query claims no version.

## The judgement of one answer

| The answer | Judgement |
|---|---|
| Lists a document that the ask did not name | ignored |
| Lists one or more documents, and the ask named all of them | honored |
| Lists no document, or no answer came | no evidence: the recorded judgement stays |

## Configuration

`YACYDHTSEARCH_PEER_RETRIAL_INTERVAL` has the default `24h`. It is the time
after which the service asks a peer again in a form it judged the peer on, when
the answers of the peer claim no software version. When the answers of a peer
claim a version, the service judges the peer again only after that version
changes. `configuration.md` gives this variable with the other variables.

## Metrics

| Metric | Labels | Answers |
|---|---|---|
| `yacydhtsearch_peer_standings_total` | `form`; `standing`: honoring, ignoring, never judged, version changed, interval passed | how many asks the recorded judgements decide, how many asks learn, and if the versions or the interval cause the new asks |
| `yacydhtsearch_peer_judgements_total` | `form`; `judged`: honored, ignored, no evidence | how much of the network honors each form |

## Lifetime

The service holds the judgements in memory. An instance that starts again holds
no judgement, and judges each peer again.
