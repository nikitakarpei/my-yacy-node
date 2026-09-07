# 12. Watch search services on dashboards

> "Can I see search service health without writing queries?"

Dashboards put the main node, crawler, and DHT search signals in ready-to-use
views for routine checks and incident response.

## What this chapter adds

- `grafana` shows the YaCy node, crawler, and DHT search metrics on
  ready-to-use dashboards.

Grafana has anonymous administrator access and listens on the local host. Add
authentication before you publish it on another address.

## Start

Start the stack:

```sh
docker compose up -d
```

## Use

Open `http://localhost:3000`. Select the YaCy node, crawler, or DHT search
dashboard.

## More information

- [YaCy node dashboard source](../../../../services/yacynode/doc/grafana-dashboard.json)
- [Crawler dashboard source](../../../../services/yacycrawler/doc/grafana-dashboard.json)
- [DHT search dashboard source](../../../../services/yacydhtsearch/doc/grafana-dashboard.json)
- [Collected metric endpoints](../../building-blocks/prometheus/prometheus.yml)
