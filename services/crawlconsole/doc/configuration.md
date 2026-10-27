# crawlconsole configuration

crawlconsole is configured entirely through environment variables.

## Console

| Variable | Default | Meaning |
|---|---|---|
| `CRAWLCONSOLE_LISTEN_ADDR` | `:8093` | Address that serves the console pages. |
| `CRAWLCONSOLE_CRAWLER_API_URL` | required | HTTP URL of the operator API of `yacycrawler`, the address set in `YACYCRAWLER_API_ADDR`. The service does not start without it. |
| `CRAWLCONSOLE_TAB_BAR` | none | JSON tab bar at the top of each page. Without it, the pages show no tab bar. |

Give each tab a `title` and an HTTP `url`, and mark the tab of the console as `current`.
An `issueTrackerUrl` is optional. For example:

```sh
CRAWLCONSOLE_TAB_BAR='{"tabs":[{"title":"Crawls","url":"http://127.0.0.1:8093","current":true}]}'
```

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `CRAWLCONSOLE_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
| `LOG_LEVEL` | `INFO` | Log level. |
