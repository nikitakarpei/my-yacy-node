# YaCy peer with search and crawling

Runs a self-hosted search engine backed by a YaCy peer. Searches combine locally crawled
pages with live web results, and opening a result crawls that page into the local corpus,
which is also shared onto the YaCy network. Searching therefore grows what the peer can
answer with.

## Services

| Service | Role |
| --- | --- |
| `searxng` | The search UI: queries the local index alongside web engines, and points every result link at `visitcrawl`. |
| `visitcrawl` | Turns an opened result into one crawl order and redirects to the page, without waiting on the order. |
| `nats` | Broker carrying crawl orders and crawled pages between services. |
| `yacycrawler` | Fetches an ordered page and turns it into text and RWI representations. |
| `renderproxy` | Proxies the page fetch through `lightpanda` so JS-rendered pages are fetched too. |
| `lightpanda` | Headless browser that renders the page for `renderproxy`; chosen over Chromium-based ones for its low memory footprint. |
| `corpustext` | Writes the text representation into the local search index. |
| `manticore` | Stores and serves the local search index. |
| `yacy-rwi-node` | The peer: shares RWI representations over the DHT and serves remote searches. |
| `smokescreen` | Egress proxy every outbound connection passes through, blocking requests to internal addresses to prevent SSRF. |
| `prometheus` | Collects metrics from every service. |

## Setup

1. Copy `.env.example` to `.env` and set `YACY_PEER_HASH`, `YACY_PEER_NAME`,
   `YACY_ADVERTISE_HOST`, `VISITCRAWL_PUBLIC_URL`, and `SEARXNG_SECRET`.
2. Start the stack: `docker compose up -d`.
3. Open `http://localhost:8080` to search.

## Extras

`extras/` holds optional add-ons, each a self-contained overlay you layer onto the base
stack. See each subfolder's README for what it adds and the exact command.
