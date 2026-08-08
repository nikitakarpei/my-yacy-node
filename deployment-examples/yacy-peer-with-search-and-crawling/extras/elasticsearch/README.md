# Elasticsearch search index

Replaces Manticore with Elasticsearch as the search index. Elasticsearch uses more memory than
Manticore; use it if you already run Elasticsearch elsewhere or need its query features.

## Use

From the example root, layer this file on top of the base stack:

```
docker compose -f compose.yml -f extras/elasticsearch/compose.yml up -d
```

This starts `elasticsearch` and `elasticsearch-metrics` alongside the base services, points
`corpustext` and `searxng` at Elasticsearch, adds the `elasticsearch` scrape job to
Prometheus, and scales `manticore` to zero so it doesn't run alongside it.
