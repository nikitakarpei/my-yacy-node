# crawlscrapebridge configuration

You configure crawlscrapebridge through environment variables.

## Brokers

| Variable | Default | Meaning |
|---|---|---|
| `CRAWL_NATS_URL` | required | NATS server the crawler publishes crawled pages on. |
| `SCRAPE_REQUEST_NATS_URL` | required | NATS server the service publishes scrape requests on. |
| `CRAWLED_PAGE_DURABLE` | `crawlscrapebridge` | Durable consumer name shared across instances. |

The service starts also while the crawled pages are unavailable. Meanwhile it logs an
error and tries again after 1 second, doubling the pause up to 30 seconds.

## Relay

| Variable | Default | Meaning |
|---|---|---|
| `CRAWLED_PAGE_RELAY_CONCURRENCY` | `4` | Crawled pages the service relays at once. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `CRAWLSCRAPEBRIDGE_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
| `LOG_LEVEL` | `INFO` | Log level. |
