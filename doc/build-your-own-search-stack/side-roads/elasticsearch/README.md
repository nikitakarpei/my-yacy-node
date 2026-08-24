# Elasticsearch instead of Manticore

Replaces Manticore with Elasticsearch as the full-text index. Elasticsearch
wants several times the memory for the same corpus, and gives you its query
language, existing tooling, and whatever cluster you already run.

Take this side road if you already operate Elasticsearch. Otherwise Manticore
holds a larger corpus on the same machine.

## Use

Layer it on any chapter that has an index, from that chapter's directory:

```sh
cd chapters/09-seeing-it
docker compose -f compose.yml -f ../../side-roads/elasticsearch/compose.yml up -d
```

It starts `elasticsearch` and `elasticsearch-metrics`, points `corpustext` and
`searxng` at Elasticsearch, adds the Elasticsearch scrape job to Prometheus, and
scales `manticore` to zero. Documents already in Manticore do not move; the new
index fills from the crawls that follow.
