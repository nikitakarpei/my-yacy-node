# yacycrawler configuration

The crawler is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `CRAWL_NATS_URL` | required | NATS server the crawler consumes crawl orders from and publishes crawled pages to. |
| `CRAWL_ORDERS_SUBJECT` | `yacy.crawl.orders` | Subject the crawler consumes orders from. |
| `CRAWL_ORDERS_DURABLE` | `yacycrawler` | Durable queue-consumer name shared across instances. |
| `PENDING_VISIT_DURABLE` | `yacycrawler-visits` | Durable queue-consumer name every instance reads pending visits from. |

The crawler publishes every page it read on the `YACY_CRAWL_PAGES` stream, which it creates
and owns. A page that states no indexing refusal goes to `crawl.page.indexable`, a page
that refuses indexing goes to `crawl.page.indexing-refused`. No name here is configurable.

To have the pages scraped, an operator binds a consumer that delivers them to the scrape
service:

```sh
nats consumer add YACY_CRAWL_PAGES scrape_request_bridge \
  --filter crawl.page.indexable --target scrape.request \
  --deliver all --ack none --replay instant
```

Bind `crawl.page.>` instead of `crawl.page.indexable` to scrape pages that refuse
indexing as well. That choice applies to every crawl the deployment runs.

## Fetching

| Variable | Default | Meaning |
|---|---|---|
| `YACYCRAWLER_FETCH_PROXY_URL` | required | Egress proxy every outbound fetch passes through. |
| `YACYCRAWLER_FETCH_PROXY_DIAL_MODE` | `tunnel` | How fetches reach the egress proxy: `tunnel` (HTTP CONNECT) or `absolute-url` (plain absolute-URL requests, for proxies that refuse CONNECT). |
| `YACYCRAWLER_MAX_BODY_BYTES` | `2097152` | Largest response body accepted; larger is disposed. |
| `YACYCRAWLER_FETCH_DEADLINE` | `30s` | Deadline for a single fetch. |
| `YACYCRAWLER_FETCH_CONCURRENCY` | `4` | Concurrent fetches on one instance. |
| `YACYCRAWLER_FETCH_USER_AGENT` | `yacycrawler (+https://yacy.net)` | User-Agent sent with every fetch. |
| `YACYCRAWLER_RECRAWL_GRACE` | `1h` | Minimum time between visits to the same URL. `0` disables suppression. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `YACYCRAWLER_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
| `LOG_LEVEL` | `INFO` | Log level. |
