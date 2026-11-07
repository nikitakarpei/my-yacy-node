# pagerwi configuration

The service is configured entirely through environment variables.

## Page offers

| Variable | Default | Meaning |
|---|---|---|
| `SCRAPE_PAGE_OFFER_NATS_URL` | required | NATS server the service takes offered pages from. |
| `SCRAPE_PAGE_OFFER_DURABLE` | `pagerwi` | Durable consumer name shared across service instances. |
| `SCRAPE_PAGE_OFFER_INTAKE_CONCURRENCY` | `4` | Offered pages the service works on at once. |

## RWIs

| Variable | Default | Meaning |
|---|---|---|
| `PAGE_RWI_NATS_URL` | required | NATS server that holds the RWI stream. |
| `PAGE_RWI_MAX_BYTES` | `1GB` | Total bytes the RWI stream keeps. |
| `PAGE_RWI_PARTITION_EXPONENT` | `4` | Partition exponent of the YaCy network. It must be `YACY_DISTRIBUTION_PARTITION_EXPONENT` of each node that reads the RWIs. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `PAGERWI_PROVISION_STREAMS` | `true` | Create the RWI stream and the page offer consumer, or set `false` to use those that exist. |
| `PAGERWI_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
| `LOG_LEVEL` | `INFO` | Minimum structured log level. |
