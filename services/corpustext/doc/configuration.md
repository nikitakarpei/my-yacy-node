# corpustext configuration

The indexer is configured entirely through environment variables.

## Broker

| Variable | Default | Meaning |
|---|---|---|
| `NATS_URL` | required | NATS server the indexer consumes crawled pages from. |
| `NATS_CRAWLED_PAGE_SUBJECT` | `yacy.crawl.page.text` | Subject the indexer consumes crawled pages from. |
| `NATS_CRAWLED_PAGE_DURABLE` | `corpustext` | Durable queue-consumer name shared across instances. |

## Indexing

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSTEXT_CONCURRENCY` | `4` | Documents indexed concurrently. |
| `SEARCH_INDEX_ENGINE` | required | Which search index to write to: `elasticsearch` or `manticore`. |
| `CORPUSTEXT_LANGUAGES` | empty | Languages that get their own index, separated by commas. |

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

## Index names

Each index name has the form `<base>_v<version>_<language>`. The base name comes from
`ELASTICSEARCH_INDEX` or `MANTICORE_TABLE`. The indexer stops at startup when it does
not accept the base name or a language, and reports the prefix `<base>_v<version>` in
its startup log. A search client reads all the indexes through this prefix.

After a change of `CORPUSTEXT_LANGUAGES` or of the schema version, crawl the pages
again or reindex them.

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSTEXT_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
