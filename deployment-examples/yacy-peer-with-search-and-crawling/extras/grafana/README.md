# Grafana dashboards

Adds Grafana, pre-provisioned with a Prometheus datasource and dashboards for the peer and
crawler. Prometheus already collects the metrics; this only adds a UI to view them.

## Use

From the example root, layer this file on top of the base stack:

```sh
docker compose -f compose.yml -f extras/grafana/compose.yml up -d
```

Open `http://localhost:3000`. No login is required.
