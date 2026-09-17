# Cross-check honoring

Status: proposal.

A cross-check asks a peer which of the named documents it holds for a word. A
YaCy peer ignores `urls=` and answers with every document it holds for the
word, so the cross-check round is off today. This proposal turns it on and
learns from each answer which peers honor a cross-check, so that only those
peers are cross-checked. A peer declares nothing. Its answer is the evidence.

The software version a peer publishes in its seed decides when the evidence is
stale. A peer is observed again only when its seed carries another version. So
a YaCy release that honors `urls=` is picked up without an operator, and a peer
that ignores it is not asked again every few hours for nothing. The version is
a claim. It decides only when to observe, never whether to ask.

## Units

- `crosscheckhonoring`, new. Owns, per peer and per cross-check form, the
  judgement of its last cross-check answer and the software version its seed
  carried at that answer. The forms are the named documents of today and the
  filter of `cross-check-by-filter.md`. Judges each answered cross-check ask.
- `peerdirectory`, changed. `KnownPeer` carries the software version from the
  seed, refreshed at each seedlist read like the addresses.
- `wordjoined`, changed. Cross-checks only the peers that honor the form or are
  on trial for it. `YACYDHTSEARCH_ASKS_FOR_CROSS_CHECKED_DOCUMENTS` stays for
  the rollout and goes away after it.
- The record lives in memory per instance. Sharing it through NATS beside the
  probe answers comes later.

## Boundaries

- Spread to honoring: the peers of one word that may be cross-checked in one
  form, from the peers that did not list all they hold. After the round, each
  answered ask, for judgement.
- Honoring to the directory: the software version of one peer.
- Honoring to the observer: each judgement, and each trial and why it started.

## Rule of one peer and one form

| Record of the peer | The cross-check |
|---|---|
| Never observed | on trial: asked once, in the listing-only form |
| Honored at version `v`, seed still at `v` | asked |
| Ignored at version `v`, seed still at `v` | not asked |
| Observed at version `v`, seed now at another version | on trial again |
| Observed, seed carries no version | on trial again after the retrial interval |

An answer that names a document outside the ask judges the peer as ignoring the
form. An answer whose every document is in the ask judges it as honoring. An
empty answer, or no answer, is no evidence, and the record stays. The listing of
a peer that ignores the form still holds documents it holds for the word, and
the join keeps them, as today.

## Configuration

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_CROSS_CHECK_RETRIAL_INTERVAL` | `24h` | Time after which a peer whose seed carries no software version is cross-checked again to observe it. A peer whose seed carries a version is observed again only when that version changes. |

## Metrics

New, published from `crosscheckhonoring` at zero from startup:

| Metric | Labels | Answers |
|---|---|---|
| `yacydhtsearch_cross_check_judgements_total` | `form`; `judged`: honored, ignored, no evidence | how much of the network honors each form |
| `yacydhtsearch_cross_check_records` | `form`; `record`: honored, ignored, on trial | how many peers are still on trial, so how many calls are still spent to learn |
| `yacydhtsearch_cross_check_retrials_total` | `form`; `because`: version changed, interval passed | whether versions or the interval drive the retrials |

Read beside them the found-only-by-cross-checking ratio of the word joined
spread for what the honoring peers add to recall, and
`yacydhtsearch_peer_calls_total` for the calls the trials cost.

## Rollout

Turn the cross-check round on with the honoring in place. The first query that
cross-checks a peer pays one trial call on it. Each judgement then holds until
the version of the peer changes. Compare the found-only-by-cross-checking ratio
and the spread duration with the period before, and then remove the flag.
