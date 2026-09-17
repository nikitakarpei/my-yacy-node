# Replica asks

Status: implemented.

The replica asks ask, for each word partition, its replicas in turn until the
partition settles, and end when every partition has settled. Hedging decides
when the next replica is asked. Coverage decides when the asks end.

## Units

- `replicaasks`, beside `peerasks`. Owns the asks put to the replicas of one
  word partition and the rule that settles the partition. Serves the
  `ReplicaAsks` port of `wordjoined` and the `PeerAsks` port of `peermatched`.
  The cross-check round and the metadata round are not replica asks.
- `hedgedelays/constant`. The hedge delay of a peer, one value from config.
  `hedgedelays/answerlatency`, later, derives it from the peer's answered calls.
- `peercallwire`, unchanged in its calls. It reports to its observer the wait for
  an in-flight slot, the taken slot, and a call cancelled before the peer answered.
- `peerasks`. `MatchedAndHeldDocumentsAsk` and `MatchedDocumentsAsk` carry the
  partition the asked peer is a replica of, from `ChosenPeer.Partition`.

## Boundaries

- Spread to replica asks: asks in, answers out. The ask order within one
  partition is the replica order peer choice produces.
- Replica asks to wire: one ask, one answer, and whether the peer answered.
- Replica asks to the `HedgeDelay` port: the delay for one peer.
- Replica asks to the observer: what settled each partition, what put each ask,
  and how the asks of one spread call ended.

## Rule of one word partition

A word partition is covered when `n` of its replicas have listed documents for
the word. `n` is the replicas covering a partition. With `n` of 1 the first
listing covers the partition. With `n` equal to the network redundancy every
replica is asked and their listings are joined.

| Event | The partition |
|---|---|
| The asks start | puts its first `n` replicas |
| An answer lists documents or counts more than zero | one listing; at `n` listings the partition is covered and cancels its outstanding calls |
| An answer holds nothing for the word | puts the next replica at once |
| Peer refused, unreachable, or unreadable | puts the next replica at once |
| A call outstanding longer than the hedge delay | puts the next replica, keeps the first; the first listing wins |
| No replica left | settles as exhausted, keeps its answers |
| Deadline of the asks | settles as deadline, keeps its answers |

The hedge timer starts at the put, so a wait for an in-flight slot counts.
Puts per partition never exceed the redundancy.

## Configuration

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_HEDGE_DELAY` | `500ms` | Time a replica call stays unanswered before the next replica of its word partition is asked. |
| `YACYDHTSEARCH_REPLICAS_COVERING_A_PARTITION` | the network redundancy | Replicas that must list documents for one word partition before the search stops asking its other replicas. `1` takes the first listing. The redundancy asks every replica. A value above the redundancy stops the service from starting. |

## Metrics

New, published from `replicaasks` at zero from startup:

| Metric | Labels | Answers |
|---|---|---|
| `yacydhtsearch_replica_asks_duration_seconds` | `ended_by`: coverage, deadline | how long the replica asks of one spread call take, and how often the deadline ends them |
| `yacydhtsearch_word_partitions_total` | `settled_by`: first, hedge, after empty answer, after failure, exhausted, deadline | what settles a partition, and how often a hedge does it |
| `yacydhtsearch_replica_asks_total` | `put_as`: first, hedge, after empty answer, after failure | calls hedging adds or saves against the redundancy per partition |
| `yacydhtsearch_word_partition_documents_listed` | none | documents listed per settled partition, the recall signal for `n` |
| `yacydhtsearch_peer_calls_waiting_for_a_slot` | none | whether the in-flight limit binds under hedging, published by the peer call metrics |

`yacydhtsearch_peer_calls_total` counts the calls that coverage cancelled under
the outcome `cancelled`. `yacydhtsearch_peer_call_duration_seconds` helps to set
the hedge delay.

## Rollout

Land with `n` equal to the redundancy. Run for a period, then set `n` to 1 for a
period of the same length. Compare `yacydhtsearch_word_partition_documents_listed`
for recall and `yacydhtsearch_replica_asks_duration_seconds` for latency, and set
the default from that. A cancelled call still costs the peer its work and its
rate-limit slot; `yacydhtsearch_peer_calls_total` shows the load per peer.
