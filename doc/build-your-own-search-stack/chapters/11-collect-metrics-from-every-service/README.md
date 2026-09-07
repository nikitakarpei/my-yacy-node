# 11. Collect search service metrics

> "How do I know when a service falls behind?"

Stored metrics show failures, work rates, and queue backlogs. They let you find
a stalled service before missing work becomes visible in search.

## What this chapter adds

- `prometheus` keeps a history of service metrics so you can query changes over
  time.
- `nats-metrics` reports how much crawl and indexing work is waiting.

## Start

Start the stack:

```sh
docker compose up -d
```

## Use

Stop an indexer:

```sh
docker compose stop corpustext
```

Submit a crawl with the
[chapter 3 command](../03-give-your-peer-a-crawler#use). Open
`http://localhost:9099` and query `nats_consumer_num_pending`. Confirm that
pending work for `corpustext` increases, then restart it:

```sh
docker compose start corpustext
```

## More information

- [Collected metric endpoints](../../building-blocks/prometheus/prometheus.yml)
