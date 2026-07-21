# corpusmarkdown configuration

The service is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `NATS_URL` | required | NATS server the service consumes crawled pages from. |
| `NATS_CRAWLED_PAGE_SUBJECT` | `yacy.crawl.page.markdown` | Subject the service consumes crawled page markdown from. |
| `NATS_CRAWLED_PAGE_DURABLE` | `corpusmarkdown` | Durable queue-consumer name shared across instances. |

## Storage

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSMARKDOWN_CONCURRENCY` | `4` | Pages stored concurrently. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSMARKDOWN_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
