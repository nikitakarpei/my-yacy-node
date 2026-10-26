# yacypeer configuration

You configure yacypeer through environment variables.

## Peer

| Variable | Default | Meaning |
|---|---|---|
| `YACY_ADVERTISE_HOST` | learned from peers | Public IP address or DNS name that other peers use to reach the peer. Required when you set `YACYPEER_EGRESS_PROXY_URL`. |
| `YACYPEER_SEEDLIST_URLS` | seedlists of `freeworld` | Comma-separated seedlists of the YaCy network. Required for a different network. |
| `YACYPEER_NETWORK_NAME` | `freeworld` | YaCy network the peer joins. |
| `YACYPEER_DATA_DIR` | `data` | Directory that holds the local data of the peer. |
| `YACYPEER_OPS_ADDR` | `127.0.0.1:9090` | Address that serves `/metrics`, `/metrics/<service>`, the dashboards at `/dashboards`, and Prometheus range queries at `/api/v1/query_range`. |
| `YACYPEER_METRICS_RETENTION` | `168h` | How long yacypeer keeps the metrics that the dashboards show. |
| `YACYPEER_EGRESS_PROXY_URL` | built-in proxy | HTTP proxy for all outbound requests. It must accept `CONNECT`. |
| `LOG_LEVEL` | `INFO` | Log level. The `service` field names the service. |

The built-in proxy refuses requests to loopback and private addresses. If you set
`YACYPEER_EGRESS_PROXY_URL`, yacypeer does not start the built-in proxy, and your proxy
must refuse these requests.

Only the local host can reach yacypeer, except on the peer address of yacynode. To let
other hosts reach an address, set it, such as `YACYDHTSEARCH_LISTEN_ADDR=:8080` or
`YACYPEER_OPS_ADDR=:9090`. In a container, you must do this to reach the address from
outside the container.

## Services

yacypeer runs [yacynode](../../../services/yacynode/doc/configuration.md),
[yacydhtsearch](../../../services/yacydhtsearch/doc/configuration.md),
[corpustext](../../../services/corpustext/doc/configuration.md),
[yacycrawler](../../../services/yacycrawler/doc/configuration.md),
[pagescrape](../../../services/pagescrape/doc/configuration.md),
[crawlconsole](../../../services/crawlconsole/doc/configuration.md),
[crawlscrapebridge](../../../services/crawlscrapebridge/doc/configuration.md), and
[visitcrawl](../../../services/visitcrawl/doc/configuration.md). Each service
reads the settings in its own configuration and serves its own endpoints.

To set a value for one service only, put the service name and `__` before the setting,
such as `CORPUSTEXT__CORPUSTEXT_LANGUAGES`. This value has priority over the setting
without a prefix.

yacypeer wires the settings that connect the services. It serves their metrics itself, so
it does not read their ops settings. If you set one of these settings, yacypeer does not
start and names the setting.
