# pagescrape configuration

The service is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `SCRAPE_NATS_URL` | unset | NATS server that holds scrape requests and page offers. When it is unset, the service serves its own NATS server. |
| `PAGESCRAPE_NATS_ADDR` | `127.0.0.1:4222` | Address of the NATS server the service serves. Other services connect to this address. |
| `PAGESCRAPE_NATS_STORE_DIR` | `./pagescrape-nats` | Directory where the NATS server the service serves keeps its streams. |
| `SCRAPE_REQUEST_DURABLE` | `pagescrape` | Durable consumer name shared across service instances. |
| `SCRAPE_REQUESTS_KEPT` | `100000` | Unfinished requests, deferred ones included, the request stream holds. A full stream refuses new requests. |
| `SCRAPE_PAGE_OFFER_MAX_BYTES` | `1GB` | Total bytes the page offer stream keeps. |

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
| `SCRAPE_DEFERRAL_WINDOW` | `6h` | Time from the arrival of a request after which a deferral fails it. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `PAGESCRAPE_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
| `LOG_LEVEL` | `INFO` | Minimum structured log level. |
