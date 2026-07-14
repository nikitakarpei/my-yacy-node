# yacycrawler configuration

The crawler is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `NATS_URL` | required | NATS server the crawler connects to. |
| `NATS_ORDERS_SUBJECT` | `yacy.crawl.orders` | Subject the crawler consumes orders from. |
| `NATS_ORDERS_DURABLE` | `yacycrawler` | Durable queue-consumer name shared across instances. |
| `NATS_PAGE_RWI_SUBJECT` | `yacy.crawl.page.rwi` | Subject the rwi representation publishes to. |
| `NATS_PAGE_RWI_MAX_MSGS` | `1024` | Bound on the rwi stream. |
| `NATS_PAGE_TEXT_SUBJECT` | `yacy.crawl.page.text` | Subject the text representation publishes to. |
| `NATS_PAGE_TEXT_MAX_MSGS` | `1024` | Bound on the text stream. |
| `NATS_PAGE_MARKDOWN_SUBJECT` | `yacy.crawl.page.markdown` | Subject the markdown representation publishes to. |
| `NATS_PAGE_MARKDOWN_MAX_MSGS` | `1024` | Bound on the markdown stream. |

## Fetching

| Variable | Default | Meaning |
|---|---|---|
| `YACYCRAWLER_PROXY_URL` | required | Egress proxy every outbound fetch passes through. |
| `YACYCRAWLER_PROXY_DIAL_MODE` | `tunnel` | How fetches reach the egress proxy: `tunnel` (HTTP CONNECT) or `absolute-url` (plain absolute-URL requests, for proxies that refuse CONNECT). |
| `YACYCRAWLER_MAX_BODY_BYTES` | `2097152` | Largest response body accepted; larger is disposed. |
| `YACYCRAWLER_FETCH_DEADLINE` | `30s` | Deadline for a single fetch. |
| `YACYCRAWLER_FETCH_CONCURRENCY` | `4` | Concurrent fetches within a single run. |
| `YACYCRAWLER_CONTENT_TYPES` | all | Comma-separated media types to crawl. Empty crawls every supported type; a list that matches none fails startup. |
| `YACYCRAWLER_USER_AGENT` | `yacycrawler (+https://yacy.net)` | User-Agent sent with every fetch. |

## Run limits

| Variable | Default | Meaning |
|---|---|---|
| `YACYCRAWLER_RUN_PAGE_BUDGET` | `1000` | Pages a single run may fetch before it stops. |
| `YACYCRAWLER_FRONTIER_CAP` | `10000` | Largest frontier a single run may hold. |

## Representations

Each enabled representation of a crawled page is published to its own stream.

| Variable | Default | Meaning |
|---|---|---|
| `YACYCRAWLER_RWI_OUTPUT_ENABLED` | `true` | Publish page references and postings. |
| `YACYCRAWLER_TEXT_OUTPUT_ENABLED` | `false` | Publish page content as text. |
| `YACYCRAWLER_MARKDOWN_OUTPUT_ENABLED` | `false` | Publish page content as markdown. |

At least one representation must be enabled, or startup fails. A representation that cannot be
derived from a page's extracted body is skipped for that page.

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `YACYCRAWLER_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
| `LOG_LEVEL` | `INFO` | Log level. |
