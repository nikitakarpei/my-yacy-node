# yacycrawler configuration

The crawler is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `CRAWL_NATS_URL` | required | NATS server the crawler consumes crawl orders from and publishes crawled pages to. |
| `CRAWL_ORDERS_SUBJECT` | `yacy.crawl.orders` | Subject the crawler consumes orders from. |
| `CRAWL_ORDERS_DURABLE` | `yacycrawler` | Durable queue-consumer name shared across instances. |

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

## Crawl state

| Variable | Default | Meaning |
|---|---|---|
| `CRAWL_STATE_DIR` | `./crawl-state` | Directory that keeps the crawls, the frontier, and the state of each page visit. |
| `CRAWL_STATE_PEBBLE_BLOCK_CACHE` | `64MB` | Memory that caches read blocks of the crawl state. |
| `CRAWL_STATE_PEBBLE_MEMTABLE_SIZE` | `16MB` | Memory that holds recent writes before they go to disk. |
| `CRAWL_STATE_PEBBLE_COMPACTION_CONCURRENCY` | `1` | Background compactions that run at the same time. |
| `CRAWL_STATE_PEBBLE_OPEN_FILE_LIMIT` | `1000` | Files the crawl state keeps open. |
| `CRAWL_STATE_PEBBLE_WRITE_CONCURRENCY` | `16` | Writes to the crawl state that run at the same time. |

One crawler instance uses one directory. Instances do not share crawl state, so the
instance that accepts an order crawls it. The crawler does not start when it cannot open
the directory.

## Fetching

| Variable | Default | Meaning |
|---|---|---|
| `YACYCRAWLER_FETCH_PROXY_URL` | required | Egress proxy every outbound fetch passes through. |
| `YACYCRAWLER_FETCH_PROXY_DIAL_MODE` | `tunnel` | How fetches reach the egress proxy: `tunnel` (HTTP CONNECT) or `absolute-url` (plain absolute-URL requests, for proxies that refuse CONNECT). |
| `YACYCRAWLER_MAX_BODY_BYTES` | `2097152` | Largest response body accepted; a larger page is not published. |
| `YACYCRAWLER_FETCH_DEADLINE` | `30s` | Deadline for a single fetch. |
| `YACYCRAWLER_FETCH_CONCURRENCY` | `4` | Concurrent fetches on one instance. |
| `YACYCRAWLER_FETCH_USER_AGENT` | `yacycrawler (+https://yacy.net)` | User-Agent sent with every fetch. |
| `YACYCRAWLER_RECRAWL_GRACE` | `1h` | Minimum time between visits to the same URL. `0` disables suppression. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `YACYCRAWLER_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
| `LOG_LEVEL` | `INFO` | Log level. |
