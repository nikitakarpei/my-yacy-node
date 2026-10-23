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
| `SEARCH_INDEX_ENGINE` | `bleve` | Which search index to use: `bleve`, `elasticsearch` or `manticore`. |
| `CORPUSTEXT_LANGUAGES` | empty | Languages that get their own index, separated by commas: `en`, `de`, `fr`, `ru`. |

When `SEARCH_INDEX_ENGINE` is `bleve`:

| Variable | Default | Meaning |
|---|---|---|
| `BLEVE_DIRECTORY` | `search-index` | Directory that holds the Bleve indexes. |

The service creates the directory when it is missing. A relative path starts at the
working directory. The working directory of the image is `/var/lib/corpustext`. Mount a
volume there to keep the indexes. Only one instance can use a directory.

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

The service answers searches also while the broker is unavailable. Meanwhile it logs an
error and tries again after 1 second, doubling the pause up to 30 seconds.

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `CORPUSTEXT_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
