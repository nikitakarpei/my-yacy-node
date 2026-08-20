# corpusrecall configuration

The service is configured entirely through environment variables.

## Startup state

The service creates no NATS state. Start the service that provisions each part below
first.

| State | Provisioned by | Needed |
|---|---|---|
| Redirect resolution bucket | yacycrawler | at startup |
| Disposed pages bucket | yacycrawler | at startup |
| Page markdown bucket | corpusmarkdown | at startup |
| Orders stream | yacycrawler | on each recall |

The service stops with an error if a bucket does not exist at startup. A recall fails
if the orders stream does not exist.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `CRAWL_NATS_URL` | required | NATS server the service places crawl orders on. |
| `PAGE_MARKDOWN_NATS_URL` | required | NATS server the service reads the page markdown from. |
| `NATS_ORDERS_SUBJECT` | `yacy.crawl.orders` | Subject the service publishes on-demand crawl orders to. |

## Retrieval

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSRECALL_LISTEN_ADDR` | `:8092` | Address serving the gRPC recall API. |
| `CORPUSRECALL_RECALL_LIMIT` | `30s` | Time a request waits for a representation before reporting its kind unavailable. |
| `CORPUSRECALL_POLL_INTERVAL` | `500ms` | Interval between corpus reads while awaiting a representation. |
| `CORPUSRECALL_MAX_IN_FLIGHT` | `256` | Requests admitted at once; further requests are rejected until one completes. |
| `CORPUSRECALL_MAX_RESPONSE_BYTES` | `4194304` | Largest single representation returned. |

`CORPUSRECALL_MAX_RESPONSE_BYTES` bounds the memory one request uses. A representation
larger than the limit is not returned. The service reports its kind unavailable
immediately, on each request, as it does for a page that is not crawled.

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSRECALL_OPS_ADDR` | `:9092` | Address serving `/metrics`. |
