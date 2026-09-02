# 11. Watch the peer and crawler on dashboards

> "Can I see peer and crawler health without writing queries?"

Dashboards put the main peer and crawler signals in one view for routine checks
and incident response.

## What this chapter adds

- `grafana` shows the YaCy node and crawler metrics on ready-to-use dashboards.

Grafana has anonymous administrator access and listens on the local host. Add
authentication before you publish it on another address.

## Start

Start the stack:

```sh
docker compose up -d
```

## Use

Open `http://localhost:3000`. Select the YaCy node or crawler dashboard.

## More information

- [YaCy node dashboard source](../../../../services/yacynode/doc/grafana-dashboard.json)
- [Crawler dashboard source](../../../../services/yacycrawler/doc/grafana-dashboard.json)
- [Collected metric endpoints](../../building-blocks/prometheus/prometheus.yml)
- [Use Elasticsearch instead of Manticore](../../side-roads/elasticsearch)
