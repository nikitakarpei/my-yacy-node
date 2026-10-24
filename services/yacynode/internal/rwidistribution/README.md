# RWI distribution

This package sends each RWI posting this node stores to its responsible peers:
the peers the DHT makes responsible for it. With handoff enabled, it deletes
the postings that enough responsible peers hold.

## Terms

- Ring partition: one of the equal parts of the DHT ring. The first bits of a
  URL hash select the ring partition of a posting. The partition exponent sets
  how many parts there are.
- Word partition: one word in one ring partition. Its postings are the postings
  of that word whose URLs are in that partition. They have the same DHT
  position and the same responsible peers.
- Holder: a peer that accepted all postings of a word partition.
- Missing holder: a responsible peer that is not a holder.
- Undelivered posting: a new or changed posting that not all its responsible
  peers accepted.
- Arc: a stretch of the ring where every position has the same responsible
  peers. These peers are the window of the arc.

## Behavior

- Each word partition has a due time. A new posting makes its word partition
  due when a worker next takes its arc, unless the word partition is in retry.
- Workers send batches. A batch holds the due word partitions of one arc. A
  worker takes only an arc whose window peers are not paused and not busy. When
  one of them becomes paused or busy, the worker leaves the arc and later
  continues from the same place. No part of the ring is in two batches at the
  same time, also when the peers change.
- An arc with nothing due waits for its next due time or a new posting.
- A holder gets the undelivered postings. A missing holder gets all postings of
  the word partition. It becomes a holder only when it accepts all of them.
- After the refresh interval, each holder gets all postings again. A holder that
  answers a refresh but does not accept every posting stops being a holder until
  it gets them all. A holder that declines stays a holder until it is excluded.
- When a responsible peer does not accept all postings it gets, the word
  partition goes into retry. The node sends its postings again only after the
  peer pause ends. Each next try waits double the previous wait, up to the
  refresh interval.
- A peer accepted a posting only when it also got the URL metadata of that
  posting. Only peers that accept a remote index get postings.
- A peer that declines is paused. A word partition that must go to
  that peer waits until the pause ends or the node excludes the peer.
- A peer is busy while it has as many batches in flight as fit into the work in
  flight at its usual time per batch. A word partition that must go to a busy
  peer waits, and the node does not send it to a peer farther away.
- When a peer declines or gives no answer to a number of offers in a row, the
  node excludes it for a time. An excluded peer is not a responsible peer, so
  the next peer takes its place, and it stops being a holder.
- A holder goes stale when closer peers replace it, or when the peer roster
  stops finding it credible. A missing holder takes its place.
- When the DHT makes this node responsible, this node holds one of the copies.
- The node sends nothing while it knows fewer reachable peers than the
  configured minimum. A batch records all of its results or none of them.

## Handoff

The node deletes the postings of a word partition when none of them is
undelivered and enough holders are responsible peers closer to its DHT position
than this node. The URL metadata stays until storage eviction removes it.

## Divergence from YaCy

- YaCy deletes a posting when it selects it for transfer. This node deletes a
  posting only after enough responsible peers hold it.
- YaCy keeps no record of holders. This node keeps the holders of each word
  partition and sends them its postings again after the refresh interval.
