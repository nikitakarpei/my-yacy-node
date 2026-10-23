# crawlconsole configuration

crawlconsole is configured entirely through environment variables.

## Console

| Variable | Default | Meaning |
|---|---|---|
| `CRAWLCONSOLE_LISTEN_ADDR` | `:8093` | Address that serves the console pages. |
| `CRAWLCONSOLE_CRAWLER_API_URL` | required | HTTP URL of the operator API of `yacycrawler`, the address set in `YACYCRAWLER_API_ADDR`. The service does not start without it. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `CRAWLCONSOLE_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
| `LOG_LEVEL` | `INFO` | Log level. |
