# YaCy peer with search and crawling

Runs a self-hosted search engine backed by a YaCy peer. Searches combine locally crawled
pages with live web results, and opening a result crawls that page into the local corpus,
which is also shared onto the YaCy network. Searching therefore grows what the peer can
answer with.

## Services

| Service | Role |
| --- | --- |
| `caddy` | The one address the stack is reached at, by a browser and by other peers. |
| `searxng` | The search UI: queries the local index alongside web engines, and points every result link at `visitcrawl`. |
| `visitcrawl` | Turns an opened result into one crawl order and redirects to the page, without waiting on the order. |
| `nats` | Broker carrying crawl orders and scrape requests between services. |
| `yacycrawler` | Fetches an ordered page and publishes the URL of every page it reached. |
| `renderproxy` | Proxies the page fetch through `lightpanda` so JS-rendered pages are fetched too. |
| `lightpanda` | Headless browser that renders the page for `renderproxy`; chosen over Chromium-based ones for its low memory footprint. |
| `corpustext` | Fetches each scrape request and writes its text into the local search index. |
| `manticore` | Stores and serves the local search index. |
| `yacy-rwi-node` | The peer: fetches each scrape request for its own word index, shares it over the DHT, and serves remote searches. |
| `smokescreen` | Egress proxy every outbound connection passes through, blocking requests to internal addresses to prevent SSRF. |
| `prometheus` | Collects metrics from every service. |

## Setup

1. Write `.env`, with a secret for the visit links and a secret for the search UI:

   ```sh
   cp .env.example .env
   printf 'VISITCRAWL_LINK_SECRET=%s\nSEARXNG_SECRET=%s\n' \
     "$(openssl rand -hex 32)" "$(openssl rand -hex 32)" >> .env
   ```

2. Set `YACY_ADVERTISE_HOST` in `.env` to the address other peers reach the stack at.
3. Start the stack: `docker compose up -d`.
4. Open `http://localhost:8080` to search.

To serve other people, and to let the peer take part in the YaCy network, publish that
address. Nothing else is published beyond the machine it runs on.

## Extras

`extras/` holds optional add-ons, each a self-contained overlay you layer onto the base
stack. See each subfolder's README for what it adds and the exact command.
