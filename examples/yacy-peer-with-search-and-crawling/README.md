# YaCy peer with search and crawling

Runs a self-hosted search engine backed by a YaCy peer. Searches combine locally crawled
pages with live web results, and opening a result crawls that page into the local corpus,
which is also shared onto the YaCy network. Searching therefore grows what the peer can
answer with.

## Services

| Service | Role |
| --- | --- |
| `searxng` | The search UI: queries the local index alongside web engines, and points every result link at `yacyvisitcrawl`. |
| `yacyvisitcrawl` | Turns an opened result into one crawl order and redirects to the page, without waiting on the order. |
| `nats` | Broker carrying crawl orders and crawled pages between services. |
| `yacycrawler` | Fetches an ordered page and turns it into text and RWI representations. |
| `renderproxy` | Runs the page in `lightpanda` so scripted content is fetched too. |
| `yacytextindexer` | Writes the text representation into the local search index. |
| `yacy-rwi-node` | The peer: shares RWI representations over the DHT and serves remote searches. |
| `smokescreen` | Egress proxy every outbound connection passes through. |
| `prometheus`, `grafana` | Metrics and the pipeline dashboard at `http://localhost:3000`, no login. |

## Setup

1. Copy `.env.example` to `.env` and set `YACY_PEER_HASH`, `YACY_PEER_NAME`,
   `YACY_ADVERTISE_HOST`, and `YACYVISITCRAWL_PUBLIC_URL`.
2. Pick a search-index engine below and copy its SearXNG settings to `searxng/settings.yml`,
   then set `server.secret_key`.
3. Copy `docker-compose.yml.example` to `docker-compose.yml`.
4. Start the stack: `docker compose up -d`.

## Search-index engine

Crawled pages are stored and served from either Elasticsearch (default) or Manticore. Set
the same engine in all three places, and keep exactly one `search-*.yml` include
uncommented in `docker-compose.yml`.

| Engine | `.env` | `docker-compose.yml` include | `searxng/settings.yml` source |
| --- | --- | --- | --- |
| Elasticsearch | `SEARCH_INDEX_ENGINE=elasticsearch` | `compose/search-elasticsearch.yml` | `searxng/settings.yml.elasticsearch.example` |
| Manticore | `SEARCH_INDEX_ENGINE=manticore` | `compose/search-manticore.yml` | `searxng/settings.yml.manticore.example` |
