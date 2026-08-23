# corpusmarkdown configuration

The service is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `SCRAPE_REQUEST_NATS_URL` | required | NATS server the service consumes scrape requests from. |
| `PAGE_MARKDOWN_NATS_URL` | required | NATS server that holds the page markdown bucket. |
| `NATS_SCRAPE_REQUEST_SUBJECT` | `scrape.request` | Subject the service consumes scrape requests from. |
| `NATS_SCRAPE_REQUEST_DURABLE` | `corpusmarkdown` | Durable queue-consumer name shared across instances. |

## Fetching

| Variable | Default | Meaning |
|---|---|---|
| `SCRAPE_PROXY_URL` | required | Egress proxy every page fetch goes through. |
| `SCRAPE_PROXY_DIAL_MODE` | `tunnel` | How to reach the proxy: `tunnel` or `absolute-url`. |
| `SCRAPE_USER_AGENT` | `corpusmarkdown (+https://yacy.net)` | User agent each fetch sends. |
| `SCRAPE_MAX_BODY_BYTES` | `2097152` | Largest body a fetch reads. |
| `SCRAPE_FETCH_DEADLINE` | `30s` | Time limit on one fetch. |

## Storage

| Variable | Default | Meaning |
|---|---|---|
| `SCRAPE_REQUEST_INTAKE_CONCURRENCY` | `4` | Scrape requests the service works on at once. |

## Recall

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSMARKDOWN_LISTEN_ADDR` | `:8094` | Address serving the markdown corpus gRPC contract. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSMARKDOWN_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
