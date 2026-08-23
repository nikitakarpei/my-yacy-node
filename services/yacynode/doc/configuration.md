# Configuration

The node is configured through environment variables.

| Variable | Default | Description |
| --- | --- | --- |
| `LOG_LEVEL` | `INFO` | Log verbosity: `DEBUG`, `INFO`, `WARN`, or `ERROR`. |
| `YACY_DATA_DIR` | `./data` | Where the node persists its data. |
| `YACY_PEER_ADDR` | `:8090` | Listen address for the YaCy peer protocol. |
| `YACY_OPS_ADDR` | `:9090` | Listen address for the `/metrics` endpoint. |
| `YACY_INITIAL_PEER_HASH` | _(empty)_ | The 12-character enhanced-Base64 peer hash to start with. Leave it empty to let the node generate one. |
| `YACY_PEER_NAME` | _(empty)_ | Peer name advertised to the network. Leave it empty to let the node derive a name from its peer hash. |
| `YACY_NETWORK_NAME` | `freeworld` | YaCy network to join. Only peers on the same network exchange data. |
| `YACY_SEEDLIST_URLS` | _(empty)_ | Comma-separated YaCy seedlist URLs to discover peers from. |
| `YACY_ADVERTISE_HOST` | _(empty)_ | Public IP or DNS name other peers use to reach you. Required when `YACY_SEEDLIST_URLS` is set. |
| `YACY_ADVERTISE_PORT` | _(the `YACY_PEER_ADDR` port)_ | Port other peers use to reach you. |
| `YACY_ANNOUNCE_INTERVAL` | `10m` | How often to re-announce yourself to the network (e.g. `30s`, `10m`, `1h`). |
| `YACY_PEER_CONTACT_CONCURRENCY` | `16` | How many peers to contact at once within an announce cycle. |
| `YACY_KNOWN_ROSTER_CAPACITY` | `4096` | Maximum number of peers the node keeps on record. |
| `YACY_REACHABLE_ROSTER_CAPACITY` | `256` | Maximum number of peers the node treats as reachable at once. |
| `YACY_TRUSTED_PROXIES` | _(empty)_ | Comma-separated CIDRs or IPs of reverse proxies fronting the node. Set this when running behind a reverse proxy so peers are not told the proxy's address. |
| `YACY_STORAGE_QUOTA` | `1GB` | Storage quota, as a human-readable size (e.g. `512MB`, `1GB`, `20GB`). |
| `YACY_ESCROW_POSTING_CAPACITY` | `8192` | How many inbound postings wait at once for their URL metadata. The node refuses further transfers until held postings expire. |
| `EGRESS_PROXY_URL` | _(required)_ | `http` or `https` URL of the proxy all outbound connections are routed through. |

## Scrape request intake

The node does not crawl. A separate crawl fleet publishes the URL of every page it
reaches; the node fetches each of those pages through its own proxy, derives the page's
words, and stores them as postings. Intake is off until `SCRAPE_REQUEST_NATS_URL` is set; without
it the node behaves as a pure peer.

| Variable | Default | Description |
| --- | --- | --- |
| `SCRAPE_REQUEST_NATS_URL` | _(empty)_ | NATS server scrape requests arrive from (e.g. `nats://nats:4222`). Empty disables intake. |
| `SCRAPE_REQUEST_SUBJECT` | `scrape.request` | Subject scrape requests arrive on. Must match the crawler. |
| `SCRAPE_REQUEST_DURABLE` | `yacy-node` | Durable queue-consumer name shared across nodes. |
| `SCRAPE_PROXY_URL` | _(required with intake)_ | Egress proxy every page fetch goes through. |
| `SCRAPE_PROXY_DIAL_MODE` | `tunnel` | How to reach the proxy: `tunnel` or `absolute-url`. |
| `SCRAPE_USER_AGENT` | `yacy-rwi-node (+https://yacy.net)` | User agent each fetch sends. |
| `SCRAPE_MAX_BODY_BYTES` | `2097152` | Largest body a fetch reads. |
| `SCRAPE_FETCH_DEADLINE` | `30s` | Time limit on one fetch. |
| `SCRAPE_REQUEST_INTAKE_CONCURRENCY` | `4` | Scrape requests the node works on at once. |

## Distribution

The node can offer its stored postings to the peers the DHT makes responsible for them.

| Variable | Default | Description |
| --- | --- | --- |
| `YACY_DISTRIBUTION_ENABLED` | `false` | Turns on outbound posting distribution. The node then also deletes a posting that enough closer peers hold. |
| `YACY_DISTRIBUTION_REDUNDANCY` | `3` | How many responsible peers must hold a posting before it counts as distributed. This node is one of them when the DHT makes it responsible. |
| `YACY_DISTRIBUTION_PARTITION_EXPONENT` | `4` | Ring partition exponent; must match the network's `network.unit.dht.partitionExponent`. |
| `YACY_DISTRIBUTION_POSTINGS_PER_BATCH` | `1000` | How many due postings to offer in one batch. A cycle offers batch after batch until no posting is due. |
| `YACY_DISTRIBUTION_CYCLE_INTERVAL` | `1m` | How often a cycle starts (e.g. `30s`, `1m`, `10m`). |
| `YACY_DISTRIBUTION_DRAIN_BUDGET` | `1m` | How long one cycle offers batches before it stops and waits for the next cycle. |
| `YACY_DISTRIBUTION_LONGEST_OFFER_INTERVAL` | `24h` | How long a posting with enough replicas waits after its due time before it is offered again. |
| `YACY_DISTRIBUTION_SHORTEST_OFFER_INTERVAL` | `5m` | How long a posting with too few replicas waits before it is offered again. The interval doubles on every further miss, up to `YACY_DISTRIBUTION_LONGEST_OFFER_INTERVAL`. |
| `YACY_DISTRIBUTION_RECIPIENT_COOLDOWN` | `10m` | How long a peer that did not accept an offer is passed over when new replicas are placed. |
| `YACY_DISTRIBUTION_MIN_REACHABLE_PEERS` | `32` | Fewest reachable peers the node must know before it offers postings. |
