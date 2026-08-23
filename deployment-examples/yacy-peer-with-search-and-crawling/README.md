# YaCy peer with search and crawling

Runs a self-hosted search engine on a YaCy peer. A search combines locally crawled pages
with live web results. Opening a result crawls that page into the local corpus, and the
peer shares that corpus on the YaCy network. Searching therefore grows what the peer can
answer.

## Services

| Search | Role |
| --- | --- |
| `caddy` | The single address that people search through and other peers connect to. |
| `searxng` | The search page, which answers from the local index and from web engines in one list. |
| `visitcrawl` | The link behind every result, which redirects you to the page and sends it to be crawled. |

| Index | Role |
| --- | --- |
| `manticore` | The local index that holds what the stack crawled and answers local searches. |
| `corpustext` | Puts the text of each crawled page into that index, so a query finds the page by its words. |
| `yacy-rwi-node` | The peer, which answers searches from other YaCy nodes and shares its index with them. |

| Crawl | Role |
| --- | --- |
| `nats` | Carries crawl orders and scrape requests, and holds them while a reader is down. |
| `yacycrawler` | Follows the links it finds on a page, so a crawl reaches more than the page you opened. |
| `renderproxy` | Gives every fetcher the page as a browser sees it, so a page built by scripts is indexed with its text. |
| `lightpanda` | The browser that `renderproxy` renders pages with, chosen over Chromium-based ones for its low memory use. |

| Egress | Role |
| --- | --- |
| `smokescreen` | Sends every service's outbound requests to public addresses only, so a crawled page cannot point the stack at internal ones. |

| Monitoring | Role |
| --- | --- |
| `prometheus` | Collects metrics from every service: request rates, failures, and broker backlog. |

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

To serve other people, and to let the peer join the YaCy network, publish that address.
Nothing else is published beyond the machine it runs on.

## Scrape request stream

Every page the crawler reaches becomes one scrape request in the `SCRAPE_REQUESTS`
stream. `corpustext` and `yacy-rwi-node` each read that stream at their own pace, so one
of them being down does not hold up the other.

The `scrape-requests-stream` job creates the stream and sets how many requests it keeps.
No service defends that window: a corpus that stays down for longer never scrapes the
pages that fall off. To keep more, raise `--max-msgs` in that job for a new stack, or run
`nats stream edit SCRAPE_REQUESTS` on a running one.

## Extras

`extras/` holds optional add-ons. Each one is a self-contained overlay on the base stack.
See each subfolder's README for what it adds and the exact command.
