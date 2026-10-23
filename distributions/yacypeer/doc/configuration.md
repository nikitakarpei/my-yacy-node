# yacypeer configuration

You configure yacypeer through environment variables.

## Peer

| Variable | Default | Meaning |
|---|---|---|
| `YACYPEER_SEEDLIST_URLS` | required | Comma-separated seedlists of the YaCy network. |
| `YACYPEER_NETWORK_NAME` | `freeworld` | YaCy network the peer joins. |
| `YACYPEER_DATA_DIR` | `data` | Directory that holds the local data of the peer. |
| `YACYPEER_OPS_ADDR` | `:9090` | Address that serves `/metrics` and `/metrics/<service>`. |
| `LOG_LEVEL` | `INFO` | Log level. The `service` field names the service. |

## Services

yacypeer runs [yacynode](../../../services/yacynode/doc/configuration.md),
[yacydhtsearch](../../../services/yacydhtsearch/doc/configuration.md),
[corpustext](../../../services/corpustext/doc/configuration.md),
[corpusmarkdown](../../../services/corpusmarkdown/doc/configuration.md),
[yacycrawler](../../../services/yacycrawler/doc/configuration.md),
[pagescrape](../../../services/pagescrape/doc/configuration.md),
[crawlconsole](../../../services/crawlconsole/doc/configuration.md), and
[crawlscrapebridge](../../../services/crawlscrapebridge/doc/configuration.md). Each service
reads the settings in its own configuration and serves its own endpoints.

The crawl console listens on `:8093`, unless you set `CRAWLCONSOLE_LISTEN_ADDR`.

To set a value for one service only, put the service name and `__` before the setting,
such as `CORPUSTEXT__CORPUSTEXT_LANGUAGES`. This value has priority over the setting
without a prefix.

yacypeer sets the settings that connect the services. It serves their metrics itself, so
it does not read their ops settings. If you set one of these settings, yacypeer does not
start and names the setting.
