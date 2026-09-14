# Configuration

The node is configured through environment variables.

## Process

| Variable | Default | Description |
| --- | --- | --- |
| `LOG_LEVEL` | `INFO` | Log verbosity: `DEBUG`, `INFO`, `WARN`, or `ERROR`. |
| `EGRESS_PROXY_URL` | _(required)_ | `http` or `https` URL of the proxy all outbound connections are routed through. |
| `YACY_PEER_ADDR` | `:8090` | Listen address for the YaCy peer protocol. |
| `YACY_OPS_ADDR` | `:9090` | Listen address for the `/metrics` endpoint. |
| `PROCESS_ENVIRONMENT_LEASE_SOCKET` | _(empty)_ | Unix socket the entrypoint waits on for the environment values a sidecar grants. Leave it empty to start the node with the environment of the container. |
| `YACY_TRUSTED_PROXIES` | _(empty)_ | Comma-separated CIDRs or IPs of reverse proxies fronting the node. Set this when running behind a reverse proxy so peers are not told the proxy's address. |

## Peer identity

Every peer publishes a seed that says who it is and where to reach it.

| Variable | Default | Description |
| --- | --- | --- |
| `YACY_INITIAL_PEER_HASH` | _(empty)_ | The 12-character enhanced-Base64 peer hash to start with. Leave it empty to let the node generate one. |
| `YACY_PEER_NAME` | _(empty)_ | Peer name advertised to the network. Leave it empty to let the node derive a name from its peer hash. |
| `YACY_NETWORK_NAME` | `freeworld` | YaCy network to join. Only peers on the same network exchange data. |
| `YACY_ADVERTISE_HOST` | _(empty)_ | Public IP or DNS name other peers use to reach you. Required when `YACY_SEEDLIST_URLS` is set. |
| `YACY_ADVERTISE_PORT` | _(the `YACY_PEER_ADDR` port)_ | Port other peers use to reach you. |

## Peer exchange

| Variable | Default | Description |
| --- | --- | --- |
| `YACY_SEEDLIST_URLS` | _(empty)_ | Comma-separated YaCy seedlist URLs to discover peers from. |
| `YACY_ANNOUNCE_INTERVAL` | `10m` | How often to re-announce yourself to the network (e.g. `30s`, `10m`, `1h`). |
| `YACY_PEER_CONTACT_CONCURRENCY` | `16` | How many peers to contact at once within an announce cycle. |
| `YACY_KNOWN_ROSTER_CAPACITY` | `4096` | Maximum number of peers the node keeps on record. |
| `YACY_REACHABLE_ROSTER_CAPACITY` | `256` | Maximum number of peers the node treats as reachable at once. |

## Storage

| Variable | Default | Description |
| --- | --- | --- |
| `YACY_DATA_DIR` | `./data` | Where the node persists its data. |
| `YACY_STORAGE_QUOTA` | `1GB` | Storage quota, as a human-readable size (e.g. `512MB`, `1GB`, `20GB`). It counts the stored keys and values, not the disk the engine uses. Give the disk headroom above it. |
| `YACY_PEBBLE_BLOCK_CACHE` | `64MB` | Memory for cached data blocks. |
| `YACY_PEBBLE_MEMTABLE_SIZE` | `8MB` | Memory a write buffer holds before the engine writes it to disk. The engine takes twice this value out of the block cache. Keep it below a quarter of `YACY_PEBBLE_BLOCK_CACHE`. |
| `YACY_PEBBLE_COMPACTION_CONCURRENCY` | `1` | How many compactions run at the same time. |
| `YACY_PEBBLE_OPEN_FILE_LIMIT` | `1000` | How many table files the engine keeps open. |
| `YACY_ESCROW_POSTING_CAPACITY` | `8192` | How many inbound postings wait at once for their URL metadata. The node refuses further transfers until held postings expire. |

## Page offer intake

The node does not crawl and does not read pages. A separate scrape service reads each page
and offers it; the node derives the page's words from the offered page and stores them as
postings. Without a page offer server it is a pure peer.

| Variable | Default | Description |
| --- | --- | --- |
| `SCRAPE_PAGE_OFFER_NATS_URL` | _(empty)_ | NATS server offered pages arrive from (e.g. `nats://nats:4222`). Empty disables intake. |
| `SCRAPE_PAGE_OFFER_DURABLE` | `yacy-node` | Durable queue-consumer name shared across nodes. |
| `SCRAPE_PAGE_OFFER_INTAKE_CONCURRENCY` | `4` | Offered pages the node works on at once. |

## Distribution

The node can offer its stored postings to the peers the DHT makes responsible for them.

| Variable | Default | Description |
| --- | --- | --- |
| `YACY_DISTRIBUTION_ENABLED` | `false` | Turns on outbound posting distribution. The node then also deletes a posting that enough closer peers hold. |
| `YACY_DISTRIBUTION_REDUNDANCY` | `3` | How many responsible peers must hold a posting before it counts as distributed. This node is one of them when the DHT makes it responsible. |
| `YACY_DISTRIBUTION_PARTITION_EXPONENT` | `4` | Ring partition exponent; must match the network's `network.unit.dht.partitionExponent`. |
| `YACY_DISTRIBUTION_POSTINGS_PER_BATCH` | `1000` | How many due postings to offer in one batch. A cycle offers batch after batch until no posting is due. |
| `YACY_DISTRIBUTION_URL_METADATA_BATCH_SIZE` | `50` | How many URL metadata records travel in one transfer to a peer. |
| `YACY_DISTRIBUTION_CYCLE_INTERVAL` | `1m` | How often a cycle starts (e.g. `30s`, `1m`, `10m`). |
| `YACY_DISTRIBUTION_DRAIN_BUDGET` | `1m` | How long one cycle offers batches before it stops and waits for the next cycle. |
| `YACY_DISTRIBUTION_LONGEST_OFFER_INTERVAL` | `24h` | How long a posting with enough replicas waits after its due time before it is offered again. |
| `YACY_DISTRIBUTION_SHORTEST_OFFER_INTERVAL` | `5m` | How long a posting with too few replicas waits before it is offered again. The interval doubles on every further miss, up to `YACY_DISTRIBUTION_LONGEST_OFFER_INTERVAL`. |
| `YACY_DISTRIBUTION_RECIPIENT_COOLDOWN` | `10m` | How long a peer that did not accept an offer is passed over when new replicas are placed. |
| `YACY_DISTRIBUTION_MIN_REACHABLE_PEERS` | `32` | Fewest reachable peers the node must know before it offers postings. |
