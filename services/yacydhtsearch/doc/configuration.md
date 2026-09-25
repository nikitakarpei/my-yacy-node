# yacydhtsearch configuration

## Process

| Variable | Default | Meaning |
|---|---|---|
| `LOG_LEVEL` | `INFO` | Log level. |
| `YACYDHTSEARCH_LISTEN_ADDR` | `:8080` | Address that serves `/yacysearch.json`. |
| `YACYDHTSEARCH_OPS_ADDR` | `:9090` | Address that serves `/metrics`. |
| `YACYDHTSEARCH_SERVE_PROFILER` | `false` | Whether the ops address also serves the Go profiler at `/debug/pprof/`. |
| `EGRESS_PROXY_URL` | required | HTTP or HTTPS proxy that peer calls and seedlist reads leave through. |
| `YACYDHTSEARCH_PAGE_READ_PROXY_URL` | the egress proxy | HTTP or HTTPS proxy that page reads leave through. |
| `YACYDHTSEARCH_PAGE_READ_PROXY_DIAL_MODE` | `tunnel` | How page reads address that proxy: `tunnel` opens a CONNECT tunnel, `absolute-url` names the whole address in the request line. |

## Network

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_NETWORK_NAME` | `freeworld` | The one YaCy network this process searches. |
| `YACYDHTSEARCH_SEEDLIST_URLS` | required | Comma-separated seedlist addresses. |
| `YACYDHTSEARCH_PARTITION_EXPONENT` | `4` | Vertical partitions of the DHT ring, as a power of two. It must match the network. |
| `YACYDHTSEARCH_NETWORK_REDUNDANCY` | `3` | How many peers hold one copy of a posting in the network. It must match the network. |

## Ranking cache

Without a NATS address each instance caches its own rankings, and a restart drops them. With one, the instances answer a repeated query from the same ranking and share the query word document amounts. An address that does not answer stops the service from starting.

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_RANKING_LIFETIME` | `2m` | Time one ranking answers a repeated query. |
| `YACYDHTSEARCH_RANKED_ITEMS_CEILING` | `50` | Most items one ranking holds. A client cannot get more than this. |
| `YACYDHTSEARCH_RANKING_CACHE_CAPACITY` | `1024` | Most rankings the cache keeps at one time. |
| `YACYDHTSEARCH_QUERY_WORD_DOCUMENT_AMOUNT_LIFETIME` | `6h` | Time the service remembers how many documents the peers hold for a query word. A query whose words all have a remembered document amount takes no sample. |
| `YACYDHTSEARCH_QUERY_WORD_DOCUMENT_AMOUNTS_CAPACITY` | `100000` | Most query words whose document amount the service remembers. |
| `YACYDHTSEARCH_NATS_URL` | in-memory | NATS address that caches rankings and query word document amounts for every instance. |

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

Peer reliability grows with the time a peer stays reachable at one address, and falls as its last answer gets older. A search asks the reliable peers first. A full directory drops its least reliable peers first.

With `YACYDHTSEARCH_NATS_URL` set, all instances keep the probe answers and the peer presence in NATS and share them. Without it, an instance keeps peer presence only while it runs.

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_PEER_PRESENCE_CONTINUITY_LIMIT` | `15m` | Most presence one probe answer can add. |
| `YACYDHTSEARCH_PROBE_ANSWER_HISTORY_KEPT_FOR` | `24h` | Time NATS keeps one probe answer. If all instances stop for longer, they lose some presence. |
| `YACYDHTSEARCH_PEER_PRESENCE_SNAPSHOT_INTERVAL` | `10m` | Time between writes of the peer presence to NATS. A longer time makes the start slower. |
| `YACYDHTSEARCH_PEER_RELIABILITY_MATURATION_DURATION` | `168h` | Presence a peer must earn for the highest reliability. More presence adds no more. |
| `YACYDHTSEARCH_PEER_RELIABILITY_STALENESS_HORIZON` | `6h` | Time after the last answer of a peer at which its reliability becomes zero. |

## Query

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_QUERY_BUDGET` | `10s` | Time one client query may take, end to end. |
| `YACYDHTSEARCH_PEER_ITEMS_CEILING` | `10` | Items this service asks one peer for. |
| `YACYDHTSEARCH_PAGES_READ_PER_QUERY` | `50` | Pages one query reads, taken from the results it puts first. |
| `YACYDHTSEARCH_PAGES_READ_PER_SITE` | `3` | Most pages of one site that one query reads. The query takes the next results in place of the others. |
| `YACYDHTSEARCH_PAGE_READ_BUDGET` | `3s` | Time the query keeps for its pages. The peer calls get the rest of the query budget. |
| `YACYDHTSEARCH_PAGE_READ_CUTOFF_PERCENT` | `90` | Percent of the pages of a query that must have an outcome before the grace starts. `100` turns the cutoff off. |
| `YACYDHTSEARCH_PAGE_READ_CUTOFF_GRACE` | `250ms` | Time the query waits for its other pages after the grace starts. |
| `YACYDHTSEARCH_URL_METADATA_LOOKUP_CUTOFF_PERCENT` | `90` | Percent of the documents that a query looks up URL metadata for that must have metadata or no open peer call before the grace starts. `100` turns the cutoff off. |
| `YACYDHTSEARCH_URL_METADATA_LOOKUP_CUTOFF_GRACE` | `250ms` | Time the query waits for the URL metadata of its other documents after the grace starts. |
| `YACYDHTSEARCH_PAGE_BYTE_CEILING` | `4194304` | Most bytes read from one page. |
| `YACYDHTSEARCH_PAGE_READ_MAX_REDIRECT_HOPS` | `3` | Most redirects followed to read one page. |
| `YACYDHTSEARCH_SNIPPET_LENGTH_CEILING` | `300` | Most characters one snippet holds. |

## Peer calls

A query asks the peers that hold its words in each partition of the ring. A query of more words asks some words only in the partitions where they can add results. Raise `YACYDHTSEARCH_PEER_CALLS_IN_FLIGHT` to put more calls at the same time, and lower it to put less load on the network. A call that waits for its turn uses the time of its query.

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_PEER_CALLS_IN_FLIGHT` | `48` | Most peer calls the service makes at the same time, over all queries. |
| `YACYDHTSEARCH_URL_METADATA_CALL_BUDGET` | `3s` | Time one peer call for URL metadata may take once it runs. |
| `YACYDHTSEARCH_SEARCH_CALL_BUDGET` | `2s` | Time one peer call for a search may take once it runs. |
| `YACYDHTSEARCH_HEDGE_DELAY` | `500ms` | Time a peer call stays unanswered before the service also asks another peer that holds the same postings. |
| `YACYDHTSEARCH_REPLICAS_COVERING_A_PARTITION` | `1` | Peers that must send documents for a word in one partition before the search stops asking the other peers that hold them. A value above the redundancy stops the service from starting. |
| `YACYDHTSEARCH_COMPOUND_WORDS_CEILING` | `4` | Most compound words asked per query. A compound word is two or three adjacent query words spelled as one (`wordpress` for `word press`). |
| `YACYDHTSEARCH_DOCUMENTS_TO_MATCH_CEILING` | `1000` | Most documents one peer call names for the peer to match. Above it, the call names none, and the answers get larger. |
| `YACYDHTSEARCH_URL_METADATA_ASK_DOCUMENTS_CEILING` | `1000` | Most documents one URL metadata call asks a peer about. A lower value puts less load on a peer, and the query can miss results. |
| `YACYDHTSEARCH_MAX_RESPONSE_BYTES` | `4194304` | Most bytes read from one peer answer or one seedlist. |
