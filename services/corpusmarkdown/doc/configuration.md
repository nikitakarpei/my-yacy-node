# corpusmarkdown configuration

The service is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `CRAWL_NATS_URL` | required | NATS server the service consumes reached pages from. |
| `PAGE_MARKDOWN_NATS_URL` | required | NATS server that holds the page markdown bucket. |
| `NATS_REACHED_PAGE_SUBJECT` | `crawl.reachedpage` | Subject the service consumes reached pages from. |
| `NATS_REACHED_PAGE_DURABLE` | `corpusmarkdown` | Durable queue-consumer name shared across instances. |

## Fetching

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSMARKDOWN_PROXY_URL` | required | Egress proxy every page fetch goes through. |
| `CORPUSMARKDOWN_PROXY_DIAL_MODE` | `tunnel` | How to reach the proxy: `tunnel` or `absolute-url`. |
| `CORPUSMARKDOWN_USER_AGENT` | `corpusmarkdown (+https://yacy.net)` | User agent each fetch sends. |
| `CORPUSMARKDOWN_MAX_BODY_BYTES` | `2097152` | Largest body a fetch reads. |
| `CORPUSMARKDOWN_FETCH_DEADLINE` | `30s` | Time limit on one fetch. |

## Storage

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSMARKDOWN_CONCURRENCY` | `4` | Pages fetched and stored concurrently. |

## Recall

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSMARKDOWN_LISTEN_ADDR` | `:8094` | Address serving the markdown corpus gRPC contract. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSMARKDOWN_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
