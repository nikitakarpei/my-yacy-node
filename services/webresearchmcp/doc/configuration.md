# webresearchmcp configuration

The service is configured entirely through environment variables.

## Search

| Variable | Default | Meaning |
|---|---|---|
| `SEARXNG_URL` | required | SearXNG instance every search goes to. |
| `SEARXNG_SEARCH_DEADLINE` | `10s` | Time limit on one search. |

## Scraping

| Variable | Default | Meaning |
|---|---|---|
| `SCRAPE_REQUEST_NATS_URL` | required | NATS server the service asks for scrapes through. |
| `SCRAPE_REQUEST_SUBJECT` | `scrape.request` | Subject the service asks for scrapes on. |
| `PAGE_MARKDOWN_NATS_URL` | required | NATS server that carries the outcome of each scrape. |
| `PAGE_FETCH_WAIT` | `10s` | Time a page call waits for the scrape it asked for. |

## Corpus

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSMARKDOWN_ADDR` | required | gRPC address of the markdown corpus the service reads pages from. |
| `CORPUSMARKDOWN_RECALL_DEADLINE` | `5s` | Time limit on one read from the corpus. |

## Answers

| Variable | Default | Meaning |
|---|---|---|
| `PAGE_FETCH_CHARACTER_LIMIT` | `5000` | Characters of markdown a page answer carries. |
| `TOOL_CALL_CONCURRENCY` | `8` | Tool calls the service works on at once. |

A call that names its own limit is answered with that limit in place of this one.

## Serving

| Variable | Default | Meaning |
|---|---|---|
| `WEBRESEARCHMCP_LISTEN_ADDR` | `:8095` | Address serving the MCP tools over HTTP. |
| `WEBRESEARCHMCP_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
