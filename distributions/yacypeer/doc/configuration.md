# yacypeer configuration

You configure yacypeer through environment variables.

## Peer

| Variable | Default | Meaning |
|---|---|---|
| `YACYPEER_SEEDLIST_URLS` | required | Comma-separated seedlists of the YaCy network. |
| `YACYPEER_NETWORK_NAME` | `freeworld` | YaCy network the peer joins. |
| `YACYPEER_DATA_DIR` | `data` | Directory that holds the data of every member. |
| `YACYPEER_OPS_ADDR` | `:9090` | Address serving `/metrics/<member>` for each member. |
| `LOG_LEVEL` | `INFO` | Log level of every member. The `member` field names the member. |

## Members

The peer runs these members. Each member reads the settings in its own configuration:

* `yacynode`: [configuration](../../../services/yacynode/doc/configuration.md)
* `yacydhtsearch`: [configuration](../../../services/yacydhtsearch/doc/configuration.md)
* `corpustext`: [configuration](../../../services/corpustext/doc/configuration.md)
* `corpusmarkdown`: [configuration](../../../services/corpusmarkdown/doc/configuration.md)

A member reads a setting from the first of these sources that has a value:

1. The value the peer wires.
2. The setting with the member name as a prefix, such as `CORPUSTEXT__CORPUSTEXT_LANGUAGES`.
3. The setting without a prefix, such as `CORPUSTEXT_LANGUAGES`. All members read it.
4. The default of the member.

You must set `EGRESS_PROXY_URL`. `yacynode` and `yacydhtsearch` send their outbound
requests through it.

## Wired settings

The peer sets these settings itself. If you set one of them, with or without a prefix,
the peer does not start and names the setting.

| Member | Settings |
|---|---|
| all | `*_OPS_ADDR` |
| `yacynode` | `SCRAPE_PAGE_OFFER_NATS_URL`, `SCRAPE_PAGE_OFFER_DURABLE`, `YACY_ACCEPT_REMOTE_INDEX`, `YACY_DATA_DIR`, `YACY_NETWORK_NAME`, `YACY_SEEDLIST_URLS` |
| `yacydhtsearch` | `YACYDHTSEARCH_NETWORK_NAME`, `YACYDHTSEARCH_SEEDLIST_URLS`, `YACYDHTSEARCH_SERVE_PROFILER` |
| `corpustext` | `SCRAPE_PAGE_OFFER_NATS_URL`, `SCRAPE_PAGE_OFFER_DURABLE`, `BLEVE_DIRECTORY` |
| `corpusmarkdown` | `SCRAPE_PAGE_OFFER_NATS_URL`, `SCRAPE_PAGE_OFFER_DURABLE`, `PAGE_MARKDOWN_DIRECTORY` |

`yacynode` takes postings only from the pages of the peer. It does not accept postings
from other peers.

## Endpoints

Each member serves its endpoints on the addresses in its own configuration. The ops
endpoint of a member is not available; use `YACYPEER_OPS_ADDR`.
