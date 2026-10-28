# yacypeer configuration

You configure yacypeer through environment variables.

## Peer

| Variable | Default | Meaning |
|---|---|---|
| `YACY_ADVERTISE_HOST` | learned from peers | Public IP address or DNS name that other peers use to reach the peer. Required when you set `YACYPEER_PEER_PROXY_URL`. |
| `YACYPEER_SEEDLIST_URLS` | seedlists of `freeworld` | Comma-separated seedlists of the YaCy network. Required for a different network. |
| `YACYPEER_NETWORK_NAME` | `freeworld` | YaCy network the peer joins. |
| `YACYPEER_DATA_DIR` | `data` | Directory that holds the local data of the peer. |
| `YACYPEER_WEB_ADDR` | `127.0.0.1:9090` | Address that serves the search page at `/`, the crawl console at `/crawlconsole/`, the dashboards at `/dashboards`, `/metrics`, and `/metrics/<service>`. |
| `YACYPEER_METRIC_HISTORY_RETENTION` | `168h` | How long yacypeer keeps the metric history that the dashboards show. |
| `YACYPEER_PEER_PROXY_URL` | none | HTTP proxy with `CONNECT` for peers and seedlists. Name a loopback or private proxy by its IP address. |
| `YACYPEER_PAGE_FETCH_PROXY_URL` | none | Proxy like `YACYPEER_PEER_PROXY_URL`, for page fetches. |
| `LOG_LEVEL` | `INFO` | Log level. The `service` field names the service. |

yacypeer does not start when `HTTP_PROXY`, `HTTPS_PROXY`, or `NO_PROXY` is set, in any case.

yacypeer makes the peer key of yacynode on its first start, and keeps it in the file
`peer-key/peer-key` of the data directory. To make the key takes some seconds. The peer
hash and the data of yacynode belong to this key, so keep a copy of the file.

Only the local host can reach yacypeer, except on the peer address of yacynode. To let
other hosts reach the web address, set it, such as `YACYPEER_WEB_ADDR=:9090`.

## Services

yacypeer runs [yacynode](../../../services/yacynode/doc/configuration.md),
[yacydhtsearch](../../../services/yacydhtsearch/doc/configuration.md),
[corpustext](../../../services/corpustext/doc/configuration.md),
[yacycrawler](../../../services/yacycrawler/doc/configuration.md),
[pagescrape](../../../services/pagescrape/doc/configuration.md),
[crawlconsole](../../../services/crawlconsole/doc/configuration.md),
[crawlscrapebridge](../../../services/crawlscrapebridge/doc/configuration.md),
[visitcrawl](../../../services/visitcrawl/doc/configuration.md), and
[searchpage](../../../services/searchpage/doc/configuration.md). Each service
reads the settings in its own configuration and serves its own endpoints.

To set a value for one service only, put the service name and `__` before the setting,
such as `CORPUSTEXT__CORPUSTEXT_LANGUAGES`. This value has priority over the setting
without a prefix.

yacypeer wires the settings that connect the services, and gives each service a free port
of the local host. It serves their metrics itself, so
it does not read their ops settings. If you set one of these settings, yacypeer does not
start and names the setting.
