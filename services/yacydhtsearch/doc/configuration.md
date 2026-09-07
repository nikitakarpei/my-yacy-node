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

One query produces one ranking, and every page of that query is cut from it. While a
ranking stays in the cache, the peers are not asked again.

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_RANKING_LIFETIME` | `2m` | Time one ranking answers a repeated query. |
| `YACYDHTSEARCH_RANKED_ITEMS_CEILING` | `50` | Most items one ranking holds. A client cannot get more than this. |
| `YACYDHTSEARCH_RANKING_CACHE_CAPACITY` | `1024` | Most rankings the cache keeps at one time. |
| `YACYDHTSEARCH_NATS_URL` | in-memory | NATS address that caches rankings for every instance. |

Without a NATS address each instance caches its own rankings, and a restart drops them. With
one, the instances answer a repeated query from the same ranking. An address that does not
answer stops the service from starting.

## Peer directory

The service probes peers to confirm that they answer. It sends searches only to
peers that answered their latest probe.

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_DIRECTORY_CAPACITY` | `4096` | Most peers the directory holds. A full directory drops the peer that answered longest ago. |
| `YACYDHTSEARCH_REFRESH_INTERVAL` | `5m` | Time between seedlist reads and probe cycles. |
| `YACYDHTSEARCH_PROBE_BUDGET` | `3s` | Time one probe of one peer address may take. |
| `YACYDHTSEARCH_PEER_SEARCH_COOLDOWN` | `5s` | Time between search requests sent to the same peer. |

## Peer selection

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_PARTITION_EXPONENT` | `4` | Vertical partitions of the DHT ring, as a power of two. It must match the network. |
| `YACYDHTSEARCH_PEER_REDUNDANCY` | `3` | Peers asked for each term in each partition. |

## Limits

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCH_QUERY_BUDGET` | `5s` | Time one client query may take, end to end. |
| `YACYDHTSEARCH_PEER_CALL_BUDGET` | `4s` | Time one call to one peer may take. |
| `YACYDHTSEARCH_PEER_CALLS_IN_FLIGHT` | `24` | Most peer calls within one query that run at the same time. |
| `YACYDHTSEARCH_MAX_RESPONSE_BYTES` | `4194304` | Most bytes read from one peer answer or one seedlist. |

## Peer search limits

YaCy peers can limit remote searches by client address. Service instances that
use the same egress proxy normally share that allowance. The peer cooldown
reduces how often this service uses that allowance on one peer.
