# 7. Knowing it is healthy

> "What happens if one of them falls behind?"

By now a dozen containers fetch, index, and trade pages without telling you
much. Each one already publishes metrics — pages fetched, pages refused,
failures, and how far behind a reader is — and nothing has been reading them.
Prometheus reads all of it on a timer, and `nats-metrics` adds what the broker
knows about its own streams.

## The change

Prometheus listens on `127.0.0.1:9099` — reachable from the machine it runs on
and nowhere else. Its scrape targets live in
`building-blocks/prometheus/prometheus.yml`.

## Try it

```sh
docker compose up -d
```

Open `http://localhost:9099` and query `nats_consumer_num_pending`. Then stop
`corpustext` for a few minutes and watch its consumer's number climb, and fall
again when you start it.

## Where you are now

You can answer any question about the stack, one PromQL expression at a time.

> "Can I see all of it at once instead?"
