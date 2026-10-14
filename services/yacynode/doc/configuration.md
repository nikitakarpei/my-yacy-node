# Configuration

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

| Variable | Default | Description |
| --- | --- | --- |
| `YACY_INITIAL_PEER_HASH` | _(empty)_ | The 12-character enhanced-Base64 peer hash to start with. Leave it empty to let the node generate one. |
| `YACY_PEER_NAME` | _(empty)_ | Peer name advertised to the network. Leave it empty to let the node derive a name from its peer hash. |
| `YACY_NETWORK_NAME` | `freeworld` | YaCy network to join. Only peers on the same network exchange data. |
| `YACY_ADVERTISE_HOST` | _(empty)_ | Public IP or DNS name other peers use to reach you. Required when `YACY_SEEDLIST_URLS` is set. |
| `YACY_ADVERTISE_PORT` | _(the `YACY_PEER_ADDR` port)_ | Port other peers use to reach you. |
| `YACY_ACCEPT_REMOTE_INDEX` | `true` | Accept the postings that other peers transfer to you. Set it to `false` when `SCRAPE_PAGE_OFFER_NATS_URL` is set: the node does not start with both sources of postings or with no source. |

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
| `YACY_PEBBLE_WRITE_CONCURRENCY` | `4` | How many write transactions run at the same time. |
| `YACY_ESCROW_POSTING_CAPACITY` | `8192` | How many inbound postings wait at once for their URL metadata. When the escrow holds this many, the node refuses transfers with unknown URLs until held postings expire. A transfer that starts below this number is held whole. |

## Page offer intake

A separate scrape service reads pages and offers them. The node stores the words of each
offered page as postings. Without a page offer server it is a pure peer.

| Variable | Default | Description |
| --- | --- | --- |
| `SCRAPE_PAGE_OFFER_NATS_URL` | _(empty)_ | NATS server offered pages arrive from (e.g. `nats://nats:4222`). Empty disables intake. |
| `SCRAPE_PAGE_OFFER_DURABLE` | `yacy-node` | Durable queue-consumer name shared across nodes. |
| `SCRAPE_PAGE_OFFER_INTAKE_CONCURRENCY` | `4` | Offered pages the node works on at once. |

## Distribution

The node can offer its stored postings to the peers the DHT makes responsible for them.
A word partition is one word in one ring partition. Its postings are the postings of that word whose URLs fall in that partition.

| Variable | Default | Description |
| --- | --- | --- |
| `YACY_DISTRIBUTION_ENABLED` | `false` | Turns on outbound posting distribution. |
| `YACY_DISTRIBUTION_HANDOFF_ENABLED` | `true` | While distribution is on, delete the postings of a word partition that enough of its responsible peers hold. Set it to `false` to keep every posting. |
| `YACY_DISTRIBUTION_REDUNDANCY` | `3` | How many responsible peers must hold a posting before it counts as distributed. This node is one of them when the DHT makes it responsible. |
| `YACY_DISTRIBUTION_PARTITION_EXPONENT` | `4` | Ring partition exponent; must match the network's `network.unit.dht.partitionExponent`. |
| `YACY_DISTRIBUTION_POSTINGS_PER_BATCH` | `1000` | A batch takes the due word partitions of one arc of the ring, at most this many word partitions. No peer gets more than this many postings in a batch, unless one word partition is larger. One transfer to a peer carries at most this many postings. |
| `YACY_DISTRIBUTION_WORKERS` | `2` | How many workers send batches at the same time. No word partition is in two batches at the same time. |
| `YACY_DISTRIBUTION_URL_METADATA_BATCH_SIZE` | `50` | How many URL metadata records travel in one transfer to a peer. |
| `YACY_DISTRIBUTION_REFRESH_INTERVAL` | `24h` | How often the holders of a word partition get all its postings again (refresh). |
| `YACY_DISTRIBUTION_PEER_PAUSE` | `10m` | The least time a peer is paused after it did not accept an offer. A paused peer gets no postings. A word partition that must go to a paused peer waits until the pause ends. A word partition that a responsible peer did not fully accept is sent again after this time, then after double the previous wait on each try, up to the refresh interval. |
| `YACY_DISTRIBUTION_FAILED_OFFERS_BEFORE_EXCLUSION` | `3` | How many offers in a row a peer can fail before the node excludes it. An offer fails when the peer gives no answer, for example after a timeout or a connection error, or when the peer refuses it. A decline is not a failure. |
| `YACY_DISTRIBUTION_PEER_EXCLUSION_DURATION` | `1h` | How long an excluded peer gets no postings. The next peer on the ring takes its place. Until the peer accepts an offer, one more failed offer excludes it again. |
| `YACY_DISTRIBUTION_PEER_WORK_IN_FLIGHT` | `1s` | How much work a peer can have in flight. The node measures how long each peer takes for all postings of one batch. A peer gets as many batches at the same time as this time divided by its usual time, and at least one. A word partition that must go to a busy peer waits. |
| `YACY_DISTRIBUTION_MIN_REACHABLE_PEERS` | `32` | Fewest reachable peers the node must know before it offers postings. |
