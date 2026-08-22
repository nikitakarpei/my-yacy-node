# corpustext configuration

The indexer is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `SCRAPE_REQUEST_NATS_URL` | required | NATS server the indexer consumes scrape requests from. |
| `NATS_SCRAPE_REQUEST_SUBJECT` | `scrape.request` | Subject the indexer consumes scrape requests from. |
| `NATS_SCRAPE_REQUEST_DURABLE` | `corpustext` | Durable queue-consumer name shared across instances. |

## Fetching

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSTEXT_PROXY_URL` | required | Egress proxy every page fetch goes through. |
| `CORPUSTEXT_PROXY_DIAL_MODE` | `tunnel` | How to reach the proxy: `tunnel` or `absolute-url`. |
| `CORPUSTEXT_USER_AGENT` | `corpustext (+https://yacy.net)` | User agent each fetch sends. |
| `CORPUSTEXT_MAX_BODY_BYTES` | `2097152` | Largest body a fetch reads. |
| `CORPUSTEXT_FETCH_DEADLINE` | `30s` | Time limit on one fetch. |

## Indexing

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSTEXT_CONCURRENCY` | `4` | Pages fetched and indexed concurrently. |
| `SEARCH_INDEX_ENGINE` | required | Which search index to write to: `elasticsearch` or `manticore`. |
| `CORPUSTEXT_LANGUAGES` | empty | Languages that get their own index, separated by commas: `en`, `de`, `fr`, `ru`. |

When `SEARCH_INDEX_ENGINE` is `elasticsearch`:

| Variable | Default | Meaning |
|---|---|---|
| `ELASTICSEARCH_URL` | required | Elasticsearch endpoint documents are indexed into. |
| `ELASTICSEARCH_INDEX` | `yacy_text` | Base name of the Elasticsearch indexes. |

When `SEARCH_INDEX_ENGINE` is `manticore`:

| Variable | Default | Meaning |
|---|---|---|
| `MANTICORE_URL` | required | Manticore endpoint documents are indexed into. |
| `MANTICORE_TABLE` | `yacy_text` | Base name of the Manticore tables. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSTEXT_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
