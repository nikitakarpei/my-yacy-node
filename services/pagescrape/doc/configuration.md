# pagescrape configuration

The service is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `SCRAPE_NATS_URL` | required | NATS server that holds scrape requests and page offers. |
| `SCRAPE_REQUEST_DURABLE` | `pagescrape` | Durable consumer name shared across service instances. |
| `SCRAPE_REQUESTS_IN_FLIGHT` | `64` | Unacknowledged scrape requests allowed across the durable consumer. |
| `SCRAPE_REQUESTS_KEPT` | `100000` | Requests and deferred requests the request stream keeps. |
| `SCRAPE_PAGE_OFFER_MAX_BYTES` | `1GB` | Total bytes the page offer stream keeps. |
| `SCRAPE_PAGE_OFFER_MAX_AGE` | `24h` | Longest time the page offer stream keeps a message. |

## Page fetches

| Variable | Default | Meaning |
|---|---|---|
| `SCRAPE_PROXY_URL` | required | HTTP or HTTPS proxy used for every page fetch. |
| `SCRAPE_PROXY_DIAL_MODE` | `tunnel` | Proxy mode: `tunnel` or `absolute-url`. |
| `SCRAPE_USER_AGENT` | `pagescrape (+https://yacy.net)` | HTTP user agent sent with every page fetch. |
| `SCRAPE_MAX_BODY_BYTES` | `2097152` | Largest response body offered; a larger body fails the scrape. |
| `SCRAPE_FETCH_DEADLINE` | `30s` | Time limit for one page fetch. |
| `SCRAPE_MAX_REDIRECT_HOPS` | `10` | Redirects followed before a page fetch fails. |

## Intake and deferral

| Variable | Default | Meaning |
|---|---|---|
| `SCRAPE_INTAKE_CONCURRENCY` | `4` | Scrape requests the service works on at once. |
| `SCRAPE_DEFERRAL_WINDOW` | `6h` | Total time a request can remain deferred before it fails. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `PAGESCRAPE_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
| `LOG_LEVEL` | `INFO` | Minimum structured log level. |
