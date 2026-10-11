# corpusmarkdown configuration

The service is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `SCRAPE_PAGE_OFFER_NATS_URL` | required | NATS server the service takes offered pages from. |
| `SCRAPE_PAGE_OFFER_DURABLE` | `corpusmarkdown` | Durable queue-consumer name shared across instances. |

## Storage

| Variable | Default | Meaning |
|---|---|---|
| `PAGE_MARKDOWN_S3_ENDPOINT` | required | HTTP address of the S3-compatible object store. |
| `PAGE_MARKDOWN_S3_REGION` | `us-east-1` | Region the object store expects in signed requests. |
| `PAGE_MARKDOWN_S3_ACCESS_KEY_ID` | required | Access key that may create the bucket and write to it. |
| `PAGE_MARKDOWN_S3_SECRET_ACCESS_KEY` | required | Secret of that access key. |
| `PAGE_MARKDOWN_BUCKET` | `yacy-page-markdown` | Bucket that holds the page markdown. |
| `SCRAPE_PAGE_OFFER_INTAKE_CONCURRENCY` | `4` | Offered pages the service works on at once. |

## Recall

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSMARKDOWN_LISTEN_ADDR` | `:8094` | Address serving the markdown corpus contract. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSMARKDOWN_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
