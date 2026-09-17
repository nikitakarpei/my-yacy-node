# Replica asks

Status: proposal.

A replica round asks, for each word partition, its replicas in turn until the
partition settles, and ends when every word partition has settled. Hedging
decides when the next replica is asked. Coverage decides when the round ends.
Today a round puts every ask at once and waits for the slowest peer.

## Units

- `replicaasks`, new, beside `peerasks`. Owns the asks put to the replicas of
  one word partition and the rule that settles the partition. Implements the
  `PeerAsks` interface of the spreads for the two replica rounds: matched and
  held documents in `wordjoined`, matched documents in `peermatched`. The
  cross-check round and the metadata round are not replica rounds and stay.
- `hedgedelays/constant`, new. The hedge delay of a peer, one value from config.
  `hedgedelays/answerlatency`, later, is a quantile of the answered-call
  durations of each peer, with the constant as the floor for a peer without history.
- `peercallwire`, changed. Loses the batch loop and exports one blocking
  single-ask call per ask kind. The in-flight semaphore moves into those calls.
- `peerasks`, changed. `MatchedAndHeldDocumentsAsk` and `MatchedDocumentsAsk`
  carry the partition the asked peer is a replica of, from `ChosenPeer.Partition`.

## Boundaries

- Spread to replica asks: asks in, answers out, method shape unchanged. The ask
  order within one partition is the replica order peer choice produces. The
  turn-major order across words in the spread goes away.
- Replica asks to wire: one ask, one answer, and whether the peer answered.
- Replica asks to the `HedgeDelay` port: the delay for one peer.
- Replica asks to the directory: a peer rests from the moment its ask is put.
  `MarkPeersChosen` before the spread goes away; a replica never asked must not rest.
- Replica asks to the observer: what settled each partition, what put each ask,
  and how each round ended.

## Rule of one word partition

A word partition is covered when `n` of its replicas have listed documents for
the word. `n` is the replicas covering a partition. With `n` of 1 the first
replica that lists documents covers the partition, and no other replica of it is
asked. With `n` equal to the network redundancy every replica is asked and their
listings are joined, as today.

| Event | The partition |
|---|---|
| Round starts | puts its first `n` replicas |
| An answer lists documents or counts more than zero | one listing; at `n` listings the partition is covered and cancels its outstanding calls |
| An answer holds nothing for the word | puts the next replica at once |
| Peer refused, unreachable, or unreadable | puts the next replica at once |
| A call outstanding longer than the hedge delay | puts the next replica, keeps the first; the first listing wins |
| No replica left | settles as exhausted, keeps its answers |
| Round deadline | settles as cut, keeps its answers |

The round ends when every partition of every word has settled, and its context
cancel ends the rest. `contextOfRound` stays: a round that ends early leaves its
time to the next round. The hedge timer starts at the put, so a wait for an
in-flight slot counts. Puts per partition never exceed the redundancy.

## Configuration

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_HEDGE_DELAY` | median of `yacydhtsearch_peer_call_duration_seconds` | Time a replica call stays unanswered before the next replica of its partition is asked. |
| `YACYDHTSEARCH_REPLICAS_COVERING_A_PARTITION` | the network redundancy | Replicas that must list documents for one word partition before the round counts it as covered and stops asking its other replicas. `1` takes the first listing. The redundancy asks every replica, as today. |

## Metrics

New, published from `replicaasks` at zero from startup:

| Metric | Labels | Answers |
|---|---|---|
| `yacydhtsearch_replica_round_duration_seconds` | `ended_by`: coverage, deadline | how long a round waits, and how often the deadline still ends it |
| `yacydhtsearch_word_partitions_total` | `settled_by`: first replica, hedge, exhausted, deadline | how often a hedge is what covers a partition |
| `yacydhtsearch_replica_asks_total` | `put_as`: first, hedge, after empty answer, after failure | calls hedging adds or saves against today's redundancy per partition |
| `yacydhtsearch_word_partition_documents_listed` | none | documents listed per settled partition, the recall signal for `n` |
| `yacydhtsearch_replica_asks_waiting_for_a_slot` | none | whether the in-flight limit binds under hedging |

Read beside them the search and spread duration histograms,
`yacydhtsearch_word_joined_spread_fully_listed_query_words_ratio`, and
`yacydhtsearch_peer_call_duration_seconds` to set the hedge delay.

## Rollout

Land with `n` equal to the redundancy. This changes only failure handling and the
round end. Run for a period, then set `n` to 1 for a period of the same length.
Compare `yacydhtsearch_word_partition_documents_listed` for recall and the round
duration for latency, and set the default from that. A cancelled call still
costs the peer its work and its rate-limit slot; `yacydhtsearch_peer_calls_total`
shows the load per peer before and after.
