# yacydhtsearchpage configuration

yacydhtsearchpage is configured entirely through environment variables.

## Page

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCHPAGE_LISTEN_ADDR` | `:8097` | Address that serves the search page. |
| `YACYDHTSEARCHPAGE_YACYDHTSEARCH_URL` | required | HTTP URL of `yacydhtsearch`, the address set in `YACYDHTSEARCH_LISTEN_ADDR`. The service does not start without it. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `YACYDHTSEARCHPAGE_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
| `LOG_LEVEL` | `INFO` | Log level. |
