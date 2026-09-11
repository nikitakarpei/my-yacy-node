# yacydhtsearch configuration

yacydhtsearch is configured entirely through environment variables.

## Search endpoint

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_LISTEN_ADDR` | `:8080` | Address that serves `/yacysearch.json`. |
| `YACYDHTSEARCH_OPS_ADDR` | `:9090` | Address that serves `/metrics`. |

## Network

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_NETWORK_NAME` | `freeworld` | The one YaCy network this process searches. |
| `YACYDHTSEARCH_SEEDLIST_URLS` | required | Comma-separated seedlist addresses. |
| `EGRESS_PROXY_URL` | required | HTTP or HTTPS proxy that every outbound request leaves through. |

## Ranking cache

One query produces one ranking, and every page of that query is cut from it. While a ranking stays in the cache, the peers are not asked again.

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_RANKING_LIFETIME` | `2m` | Time one ranking answers a repeated query. |
| `YACYDHTSEARCH_RANKED_ITEMS_CEILING` | `50` | Most items one ranking holds. A client cannot get more than this. |
| `YACYDHTSEARCH_RANKING_CACHE_CAPACITY` | `1024` | Most rankings the cache keeps at one time. |
| `YACYDHTSEARCH_NATS_URL` | in-memory | NATS address that caches rankings for every instance. |

Without a NATS address each instance caches its own rankings, and a restart drops them. With one, the instances answer a repeated query from the same ranking. An address that does not answer stops the service from starting.

## Peer directory

The service probes peers to confirm that they answer. It sends searches only to peers that answered their latest probe.
YaCy peers can limit remote searches by client address. Service instances that use the same egress proxy share that allowance, and the peer cooldown reduces how often this service uses it on one peer.

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_DIRECTORY_CAPACITY` | `4096` | Most peers the directory holds. A full directory drops the peer that answered longest ago. |
| `YACYDHTSEARCH_REFRESH_INTERVAL` | `5m` | Time between seedlist reads and probe cycles. |
| `YACYDHTSEARCH_PROBE_BUDGET` | `3s` | Time one probe of one peer address may take. |
| `YACYDHTSEARCH_PROBES_IN_FLIGHT` | `24` | Most probes of one cycle that run at the same time. |
| `YACYDHTSEARCH_PEER_CHOICE_COOLDOWN` | `5s` | Time a chosen peer rests before a search may choose it again. |

## Peer selection

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_PARTITION_EXPONENT` | `4` | Vertical partitions of the DHT ring, as a power of two. It must match the network. |

## Cross-peer words

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_WORD_JOINED_SEARCH` | `false` | Use word joined search for queries with more than one term. Queries with one term use peer matched search. |
| `YACYDHTSEARCH_RELEVANCE_RANKING` | `false` | Order the ranking by how well each result answers the query. The ranking keeps the order the peers put their results in when this is false. |

## Page reading

The service reads the page of each candidate result, and reads all these pages at the same time. It counts the query words in the text of that page, and cuts the snippet of the result from the same text. A page the service cannot read leaves its result as the peers answered it.

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_PAGES_READ_PER_QUERY` | `50` | Pages one query reads, taken from the results it puts first. |
| `YACYDHTSEARCH_PAGE_READ_BUDGET` | `3s` | Time the pages of one query may take, inside the query budget. |
| `YACYDHTSEARCH_PAGE_BYTE_CEILING` | `1048576` | Most bytes read from one page. |
| `YACYDHTSEARCH_SNIPPET_LENGTH_CEILING` | `300` | Most characters one snippet holds. |

## Limits

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_QUERY_BUDGET` | `8s` | Time one client query may take, end to end. |
| `YACYDHTSEARCH_PEER_CALLS_PER_QUERY` | `24` | Most peer calls one query puts. The query shares them over its terms. |
| `YACYDHTSEARCH_MAX_RESPONSE_BYTES` | `4194304` | Most bytes read from one peer answer or one seedlist. |
| `YACYDHTSEARCH_PEER_ITEMS_CEILING` | `10` | Items this service asks one peer for. |

Word joined search puts two rounds of peer calls. Each round stays within `YACYDHTSEARCH_PEER_CALLS_PER_QUERY`, and each peer call leaves the peer the time the query has left.
