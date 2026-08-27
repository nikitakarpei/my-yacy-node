# yacycrawler configuration

The crawler is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `CRAWL_NATS_URL` | required | NATS server the crawler consumes crawl orders from. |
| `SCRAPE_REQUEST_NATS_URL` | required | NATS server the crawler publishes scrape requests to. |
| `CRAWL_ORDERS_SUBJECT` | `yacy.crawl.orders` | Subject the crawler consumes orders from. |
| `CRAWL_ORDERS_DURABLE` | `yacycrawler` | Durable queue-consumer name shared across instances. |
| `PENDING_VISIT_DURABLE` | `yacycrawler-visits` | Durable queue-consumer name every instance reads pending visits from. |

The crawler publishes a scrape request for every page it admits on the `scrape.request`
subject of the `SCRAPE_REQUESTS` stream. Neither is configurable. The crawler does not
create that stream. An operator creates it before the crawler starts.

The crawler creates the `YACY_CRAWL_FRONTIER` stream, which holds one message per URL an
order still owes a visit, and the `YACY_VISIT_CLAIMS` and `YACY_ACCEPTED_ORDERS` buckets.
Every instance reads and writes all three, so a run continues on any instance that is up.

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
