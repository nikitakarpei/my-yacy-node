# Configuration

The node is configured through environment variables.

| Variable | Default | Description |
| --- | --- | --- |
| `LOG_LEVEL` | `INFO` | Log verbosity: `DEBUG`, `INFO`, `WARN`, or `ERROR`. |
| `YACY_DATA_DIR` | `./data` | Where the node persists its data. |
| `YACY_PEER_ADDR` | `:8090` | Listen address for the YaCy peer protocol. |
| `YACY_OPS_ADDR` | `:9090` | Listen address for the `/metrics` endpoint. |
| `YACY_PEER_HASH` | _(required)_ | The 12-character enhanced-Base64 peer hash advertised to the network. |
| `YACY_PEER_NAME` | _(required)_ | Peer name advertised to the network. |
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
| `YACY_PROXY_URL` | _(required)_ | `http` or `https` URL of the proxy all outbound connections are routed through. |

## Crawl ingest

The node does not crawl. It receives crawled pages from a separate crawl fleet over NATS JetStream and stores them as postings. Ingest is off until `NATS_URL` is set; without it the node behaves as a pure peer.

| Variable | Default | Description |
| --- | --- | --- |
| `NATS_URL` | _(empty)_ | NATS server to reach the crawl fleet (e.g. `nats://nats:4222`). Empty disables ingest. |
| `NATS_INGEST_SUBJECT` | `yacy.crawl.page.rwi` | Subject crawled batches arrive on. Must match the crawler. |
| `NATS_INGEST_DURABLE` | `yacy-node` | Durable consumer name for reading ingest batches. |

## Distribution

The node can offer its stored postings to the peers the DHT makes responsible for them.

| Variable | Default | Description |
| --- | --- | --- |
| `YACY_DISTRIBUTION_ENABLED` | `false` | Turns on outbound posting distribution. |
| `YACY_DISTRIBUTION_REDUNDANCY` | `3` | How many responsible peers must accept a posting before it counts as distributed. |
| `YACY_DISTRIBUTION_PARTITION_EXPONENT` | `4` | Ring partition exponent; must match the network's `network.unit.dht.partitionExponent`. |
| `YACY_DISTRIBUTION_POSTINGS_PER_CYCLE` | `1000` | How many due postings to offer in each cycle. |
| `YACY_DISTRIBUTION_CYCLE_INTERVAL` | `1m` | How often to drain due postings (e.g. `30s`, `1m`, `10m`). |
| `YACY_DISTRIBUTION_REFRESH_INTERVAL` | `24h` | How long a posting with sufficient replicas waits before it is offered again. |
| `YACY_DISTRIBUTION_RETRY_INTERVAL` | `5m` | How long a posting with too few replicas waits when the peer asks for no pause. |
| `YACY_DISTRIBUTION_MIN_REACHABLE_PEERS` | `32` | Fewest reachable peers the node must know before it offers postings. |
