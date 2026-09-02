# corpusmarkdown configuration

The service is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `SCRAPE_PAGE_OFFER_NATS_URL` | required | NATS server the service takes offered pages from. |
| `PAGE_MARKDOWN_NATS_URL` | required | NATS server that holds the page markdown bucket. |
| `SCRAPE_PAGE_OFFER_DURABLE` | `corpusmarkdown` | Durable queue-consumer name shared across instances. |

## Storage

| Variable | Default | Meaning |
|---|---|---|
| `SCRAPE_PAGE_OFFER_INTAKE_CONCURRENCY` | `4` | Offered pages the service works on at once. |

## Recall

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSMARKDOWN_LISTEN_ADDR` | `:8094` | Address serving the markdown corpus gRPC contract. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSMARKDOWN_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
