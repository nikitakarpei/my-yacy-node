# Use Elasticsearch instead of Manticore

Use Elasticsearch when you already operate it or need its tools. Keep Manticore
when lower resource use is more important.

## What this alternative changes

- `elasticsearch` stores and searches the full-text index.
- `elasticsearch-metrics` exposes Elasticsearch metrics to Prometheus.
- `manticore` is disabled by this Compose overlay.

## Start

Use chapter 10 or 11 because these chapters include Prometheus. Complete the
[common preparation](../../README.md#prepare-a-chapter) for that chapter.

This overlay does not move existing Manticore documents. Plan a recrawl after
the switch.

From the selected chapter directory:

```sh
docker compose -f compose.yml \
  -f ../../side-roads/elasticsearch/compose.yml up -d
```

## Use

Submit a crawl and search for its pages at `http://localhost:8080`. Open
Prometheus at `http://localhost:9099` and query `up{job="elasticsearch"}`.

## More information

- [Elasticsearch index operation](../../../../services/corpustext/doc/elasticsearch.md)
- [Full-text indexer configuration](../../../../services/corpustext/doc/configuration.md)
