# corpustext configuration

The indexer is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `SCRAPE_PAGE_OFFER_NATS_URL` | required | NATS server the indexer takes offered pages from. |
| `SCRAPE_PAGE_OFFER_DURABLE` | `corpustext` | Durable queue-consumer name shared across instances. |

## Indexing

| Variable | Default | Meaning |
|---|---|---|
| `SCRAPE_PAGE_OFFER_INTAKE_CONCURRENCY` | `4` | Offered pages the service works on at once. |
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

## Search

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSTEXT_SEARCH_ADDR` | `:8096` | Address serving `GET /search`. |

The service answers searches from the search index that `SEARCH_INDEX_ENGINE` names. It
starts to answer before it reaches the broker. While the broker or its stream is not
available, the service logs an error and tries again. The pause starts at 1 second and
doubles up to 30 seconds.

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSTEXT_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
