# yacycrawler configuration

The crawler is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `NATS_URL` | required | NATS server the crawler connects to. |
| `NATS_ORDERS_SUBJECT` | `yacy.crawl.orders` | Subject the crawler consumes orders from. |
| `NATS_ORDERS_DURABLE` | `yacycrawler` | Durable queue-consumer name shared across instances. |

Per-representation stream variables are covered under Representations.

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
| `YACYCRAWLER_RECRAWL_GRACE` | `1h` | Minimum time between visits to the same URL. `0` disables suppression. |

## Run limits

| Variable | Default | Meaning |
|---|---|---|
| `YACYCRAWLER_RUN_PAGE_BUDGET` | `1000` | Pages a single run may fetch before it stops. |
| `YACYCRAWLER_FRONTIER_CAP` | `10000` | Largest frontier a single run may hold. |

## Representations

Each enabled representation of a crawled page is published to its own stream. Every
representation `<REP>` is configured by the same three variables, where `<REP>` is its
name upper-cased:

| Variable | Default | Meaning |
|---|---|---|
| `YACYCRAWLER_PUBLISH_<REP>` | per representation | Publish this representation. |
| `NATS_PAGE_<REP>_SUBJECT` | per representation | Subject this representation publishes to. |
| `NATS_PAGE_<REP>_MAX_MSGS` | `1024` | Bound on this representation's stream. |

| Representation | Enabled | Subject | Content |
|---|---|---|---|
| `rwi` | `true` | `yacy.crawl.page.rwi` | Page references and postings. |
| `text` | `false` | `yacy.crawl.page.text` | Page content as text. |
| `markdown` | `false` | `yacy.crawl.page.markdown` | Page content as markdown. |

At least one representation must be enabled, or startup fails. Each representation accepts only
some content formats; a page whose format none of the enabled ones accepts is disposed.

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `YACYCRAWLER_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
| `LOG_LEVEL` | `INFO` | Log level. |
