# yacydhtsearchpage configuration

yacydhtsearchpage is configured entirely through environment variables.

## Page

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCHPAGE_LISTEN_ADDR` | `:8097` | Address that serves the search page. |
| `YACYDHTSEARCHPAGE_YACYDHTSEARCH_URL` | required | HTTP URL of `yacydhtsearch`, the address set in `YACYDHTSEARCH_LISTEN_ADDR`. The service does not start without it. |

## Visits

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCHPAGE_VISITCRAWL_URL` | none | HTTP URL of `visitcrawl`. Set it to send each opened result through `visitcrawl`. Leave it empty to link the results directly. |
| `VISITCRAWL_LINK_SECRET` | required with `YACYDHTSEARCHPAGE_VISITCRAWL_URL` | Secret that signs the visit links. Set it to the same value as `visitcrawl`. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCHPAGE_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
| `LOG_LEVEL` | `INFO` | Log level. |
