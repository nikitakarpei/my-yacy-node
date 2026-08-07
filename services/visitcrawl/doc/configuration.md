# visitcrawl configuration

The visit intake service is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `NATS_URL` | required | NATS server the service places crawl orders on. |
| `NATS_ORDERS_SUBJECT` | `yacy.crawl.orders` | Subject crawl orders are placed on. Must match the crawler. |

## Visit links

| Variable | Default | Meaning |
|---|---|---|
| `VISITCRAWL_LINK_SECRET` | required | Secret the link issuer signs visit links with. Must match the issuer. |

## Placement

| Variable | Default | Meaning |
|---|---|---|
| `VISITCRAWL_ORDER_TIMEOUT` | `5s` | Time bound on a single placement attempt. |
| `VISITCRAWL_MAX_IN_FLIGHT` | `256` | Concurrent placement attempts allowed before new visits are skipped. |
| `VISITCRAWL_MAX_BODY_BYTES` | `4096` | Largest request body accepted on `/visit`. |

## Crawl profile

Every placed order carries the same crawl profile, built once from these variables.

| Variable | Default | Meaning |
|---|---|---|
| `VISITCRAWL_CRAWL_SCOPE` | `domain` | One of `domain`, `wide`, `subpath`. |
| `VISITCRAWL_CRAWL_NAME` | empty | Human-readable profile name. |
| `VISITCRAWL_CRAWL_MAX_DEPTH` | `1` | Link depth the crawl follows from the visited page. |
| `VISITCRAWL_CRAWL_URL_MUST_MATCH` | match all | Regular expression a URL must match to be crawled. |
| `VISITCRAWL_CRAWL_URL_MUST_NOT_MATCH` | none | Regular expression that excludes a URL from the crawl. |
| `VISITCRAWL_CRAWL_MAX_PAGES_PER_HOST` | `100` | Pages per host the crawl may fetch; `-1` is unlimited. |
| `VISITCRAWL_CRAWL_ALLOW_QUERY_URLS` | `false` | Whether URLs with a query string may be crawled. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `VISITCRAWL_LISTEN_ADDR` | `:8091` | Address serving `/visit`. |
| `VISITCRAWL_OPS_ADDR` | `:9091` | Address serving `/metrics`. |
| `LOG_LEVEL` | `INFO` | Log level. |
