# corpustext configuration

The indexer is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `CRAWL_NATS_URL` | required | NATS server the indexer consumes crawled pages from. |
| `NATS_CRAWLED_PAGE_SUBJECT` | `yacy.crawl.page.text` | Subject the indexer consumes crawled pages from. |
| `NATS_CRAWLED_PAGE_DURABLE` | `corpustext` | Durable queue-consumer name shared across instances. |

## Indexing

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSTEXT_CONCURRENCY` | `4` | Documents indexed concurrently. |
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
