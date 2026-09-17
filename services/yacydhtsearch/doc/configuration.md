# yacydhtsearch configuration

## Process

| Variable | Default | Meaning |
|---|---|---|
| `LOG_LEVEL` | `INFO` | Log level. |
| `YACYDHTSEARCH_LISTEN_ADDR` | `:8080` | Address that serves `/yacysearch.json`. |
| `YACYDHTSEARCH_OPS_ADDR` | `:9090` | Address that serves `/metrics`. |
| `EGRESS_PROXY_URL` | required | HTTP or HTTPS proxy that every outbound request leaves through. |

## Network

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_NETWORK_NAME` | `freeworld` | The one YaCy network this process searches. |
| `YACYDHTSEARCH_SEEDLIST_URLS` | required | Comma-separated seedlist addresses. |
| `YACYDHTSEARCH_PARTITION_EXPONENT` | `4` | Vertical partitions of the DHT ring, as a power of two. It must match the network. |
| `YACYDHTSEARCH_NETWORK_REDUNDANCY` | `3` | How many peers hold one copy of a posting in the network. It must match the network. |

## Ranking cache

Without a NATS address each instance caches its own rankings, and a restart drops them. With one, the instances answer a repeated query from the same ranking. An address that does not answer stops the service from starting.

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_RANKING_LIFETIME` | `2m` | Time one ranking answers a repeated query. |
| `YACYDHTSEARCH_RANKED_ITEMS_CEILING` | `50` | Most items one ranking holds. A client cannot get more than this. |
| `YACYDHTSEARCH_RANKING_CACHE_CAPACITY` | `1024` | Most rankings the cache keeps at one time. |
| `YACYDHTSEARCH_NATS_URL` | in-memory | NATS address that caches rankings for every instance. |

## Peer directory

YaCy peers can limit remote searches by client address. Service instances that use the same egress proxy share that allowance.

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_DIRECTORY_CAPACITY` | `4096` | Most peers the directory holds. |
| `YACYDHTSEARCH_DIRECTORY_NEWCOMER_SHARE` | `0.05` | Part of a full directory given to new peers at each seedlist read. `0` admits no new peer into a full directory. |
| `YACYDHTSEARCH_REFRESH_INTERVAL` | `5m` | Time between seedlist reads and probe cycles. |
| `YACYDHTSEARCH_PROBE_BUDGET` | `3s` | Time one probe of one peer address may take. |
| `YACYDHTSEARCH_PROBES_IN_FLIGHT` | `24` | Most probes of one cycle that run at the same time. |

## Peer presence and reliability

Peer presence is the time a peer stayed reachable at one address. Each probe answer adds the time since the previous answer of that peer. Peer reliability comes from presence and from the age of the last answer. Among the peers that hold a word, a search asks the reliable peers first. A full directory drops its least reliable peers first.

With `YACYDHTSEARCH_NATS_URL` set, all instances keep the probe answers and the peer presence in NATS and share them. Without it, an instance keeps peer presence only while it runs.

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_PEER_PRESENCE_CONTINUITY_LIMIT` | `15m` | Most presence one probe answer can add. |
| `YACYDHTSEARCH_PROBE_ANSWER_HISTORY_KEPT_FOR` | `24h` | Time NATS keeps one probe answer. If all instances stop for longer, they lose the presence from the answers after the last write of the peer presence. |
| `YACYDHTSEARCH_PEER_PRESENCE_SNAPSHOT_INTERVAL` | `10m` | Time between writes of the peer presence to NATS. A starting instance reads the last write, then the answers after it, so a longer time makes the start slower. |
| `YACYDHTSEARCH_PEER_RELIABILITY_MATURATION_DURATION` | `168h` | Presence a peer must earn for the highest reliability. More presence adds no more. |
| `YACYDHTSEARCH_PEER_RELIABILITY_STALENESS_HORIZON` | `6h` | Time after the last answer of a peer at which its reliability becomes zero. |

## Query

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_QUERY_BUDGET` | `10s` | Time one client query may take, end to end. |
| `YACYDHTSEARCH_PEER_ITEMS_CEILING` | `10` | Items this service asks one peer for. |
| `YACYDHTSEARCH_PAGES_READ_PER_QUERY` | `50` | Pages one query reads, taken from the results it puts first. |
| `YACYDHTSEARCH_PAGE_READ_BUDGET` | `3s` | Time the query keeps for its pages. The peer calls get the rest of the query budget. A page that is not read leaves its result as the peers answered it. |
| `YACYDHTSEARCH_PAGE_BYTE_CEILING` | `4194304` | Most bytes read from one page. |
| `YACYDHTSEARCH_SNIPPET_LENGTH_CEILING` | `300` | Most characters one snippet holds. |

## Peer calls

A query asks the peers that hold each of its words, which is the partitions of the ring times the redundancy of the network, for every word. A query of more than one word can ask the peers of a word a second time, to find which documents of another word they also hold. Raise `YACYDHTSEARCH_PEER_CALLS_IN_FLIGHT` to put more of them at the same time, and lower it to put less load on the network. A peer call that waits for its turn keeps the time its query has left.

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_PEER_CALLS_IN_FLIGHT` | `48` | Most peer calls the service makes at the same time, over all queries. |
| `YACYDHTSEARCH_PEER_CALL_BUDGET` | `3s` | Time one peer call may take once it runs. |
| `YACYDHTSEARCH_CROSS_CHECKED_DOCUMENTS_CEILING` | `1000` | Most documents one second call asks a peer about. A lower value puts less load on a peer, and the query can miss results. |
| `YACYDHTSEARCH_MAX_RESPONSE_BYTES` | `4194304` | Most bytes read from one peer answer or one seedlist. |
