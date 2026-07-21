# corpusrecall configuration

The service is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `NATS_URL` | required | NATS server the service places crawl orders on and reads the corpus from. |
| `NATS_ORDERS_SUBJECT` | `yacy.crawl.orders` | Subject the service publishes on-demand crawl orders to. |

## Retrieval

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSRECALL_LISTEN_ADDR` | `:8092` | Address serving the gRPC recall API. |
| `CORPUSRECALL_DEADLINE` | `30s` | Time a request waits for a representation before reporting its kind unavailable. |
| `CORPUSRECALL_POLL_INTERVAL` | `500ms` | Interval between corpus reads while awaiting a representation. |
| `CORPUSRECALL_MAX_IN_FLIGHT` | `256` | Requests admitted at once; further requests are rejected until one completes. |
| `CORPUSRECALL_MAX_RESPONSE_BYTES` | `4194304` | Largest single representation returned. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSRECALL_OPS_ADDR` | `:9092` | Address serving `/metrics`. |
