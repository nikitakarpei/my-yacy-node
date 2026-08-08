# searxng-crawled-text-search configuration

The engine is configured entirely through its `engines:` entry in SearXNG's own `settings.yml` —
no environment variables.

## Enabling the engine

Add the engine's directory to SearXNG's `engines/` folder (or mount `crawled_text_search.py`
there), then add it to `settings.yml`:

```yaml
engines:
  - name: crawled text search
    engine: crawled_text_search
    shortcut: ct
    categories: general
    enable_http: true
    search_index_engine: elasticsearch
    elasticsearch_url: http://elasticsearch:9200
```

| Key | Default | Meaning |
|---|---|---|
| `search_index_engine` | required | Which search index to query: `elasticsearch` or `manticore`. |

When `search_index_engine` is `elasticsearch`:

| Key | Default | Meaning |
|---|---|---|
| `elasticsearch_url` | required | Base URL of the Elasticsearch instance to query. |
| `elasticsearch_index` | `yacy_text_v1` | Prefix of the indexes `corpustext` writes documents into. |

When `search_index_engine` is `manticore`:

| Key | Default | Meaning |
|---|---|---|
| `manticore_url` | required | Base URL of the Manticore instance to query. |
| `manticore_table` | `yacy_text_v1` | Distributed table that spans the tables `corpustext` writes documents into. |
