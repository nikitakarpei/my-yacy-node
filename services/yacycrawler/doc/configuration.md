# yacycrawler configuration

The crawler is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `CRAWL_NATS_URL` | required | NATS server the crawler consumes crawl orders from. |
| `SCRAPE_REQUEST_NATS_URL` | required | NATS server the crawler publishes scrape requests to. |
| `CRAWL_ORDERS_SUBJECT` | `yacy.crawl.orders` | Subject the crawler consumes orders from. |
| `CRAWL_ORDERS_DURABLE` | `yacycrawler` | Durable queue-consumer name shared across instances. |

The crawler publishes a scrape request for every page it admits on the `scrape.request`
subject of the `SCRAPE_REQUESTS` stream. Neither is configurable. The crawler does not
create that stream. An operator creates it before the crawler starts.

## Fetching

| Variable | Default | Meaning |
|---|---|---|
| `SCRAPE_PROXY_URL` | required | Egress proxy every outbound fetch passes through. |
| `SCRAPE_PROXY_DIAL_MODE` | `tunnel` | How fetches reach the egress proxy: `tunnel` (HTTP CONNECT) or `absolute-url` (plain absolute-URL requests, for proxies that refuse CONNECT). |
| `YACYCRAWLER_MAX_BODY_BYTES` | `2097152` | Largest response body accepted; larger is disposed. |
| `YACYCRAWLER_FETCH_DEADLINE` | `30s` | Deadline for a single fetch. |
| `YACYCRAWLER_FETCH_CONCURRENCY` | `4` | Concurrent fetches within a single run. |
| `SCRAPE_USER_AGENT` | `yacycrawler (+https://yacy.net)` | User-Agent sent with every fetch. |
| `YACYCRAWLER_RECRAWL_GRACE` | `1h` | Minimum time between visits to the same URL. `0` disables suppression. |

## Run limits

| Variable | Default | Meaning |
|---|---|---|
| `YACYCRAWLER_RUN_PAGE_BUDGET` | `1000` | Pages a single run may fetch before it stops. |
| `YACYCRAWLER_FRONTIER_CAP` | `10000` | Largest frontier a single run may hold. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `YACYCRAWLER_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
| `LOG_LEVEL` | `INFO` | Log level. |
