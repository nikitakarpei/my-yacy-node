# searchpage configuration

searchpage is configured entirely through environment variables.

## Page

| Variable | Default | Meaning |
|---|---|---|
| `SEARCHPAGE_LISTEN_ADDR` | `:8097` | Address that serves the search page and `/visit`. |
| `SEARCHPAGE_TAB_BAR` | none | JSON tab bar at the top of each page. Without it, the pages show no tab bar. |

Give each tab a `title` and a `url`: an HTTP URL, or a path on the same host such as
`/dashboards`. Mark the tab of the search page as `current`.
An `issueTrackerUrl` is optional. For example:

```sh
SEARCHPAGE_TAB_BAR='{"tabs":[{"title":"Search","url":"http://127.0.0.1:8097","current":true}]}'
```

## Sources

| Variable | Default | Meaning |
|---|---|---|
| `SEARCHPAGE_CORPUSTEXT_URL` | none | HTTP URL of the search of `corpustext`, the address set in `CORPUSTEXT_SEARCH_ADDR`. |
| `SEARCHPAGE_YACYSEARCH_URL` | none | HTTP URL of a server of `/yacysearch.json`, such as `yacydhtsearch`. |
| `SEARCHPAGE_YACYSEARCH_CALL_DEADLINE` | `12s` | Longest time that a search waits for the YaCy network. Set it above `YACYDHTSEARCH_QUERY_BUDGET`. |

The page searches only the sources that you set. Set at least one, or the service does not
start.

## Visit links

| Variable | Default | Meaning |
|---|---|---|
| `SEARCHPAGE_VISITCRAWL_URL` | none | HTTP URL of `visitcrawl`, the address set in `VISITCRAWL_LISTEN_ADDR`. |
| `VISITCRAWL_LINK_SECRET` | none | Secret that signs the visit links. Set it to the same value as `visitcrawl`. |
| `SEARCHPAGE_VISIT_LINK_LIFETIME` | `24h` | Time that a visit link stays valid. |

Set both `SEARCHPAGE_VISITCRAWL_URL` and `VISITCRAWL_LINK_SECRET`, or neither. With both, a
person opens each result at `/visit` of the search page, and the service sends the visit on to
`visitcrawl`. With neither, each result links straight to its page.

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `SEARCHPAGE_OPS_ADDR` | `:9090` | Address serving `/metrics`. |
| `LOG_LEVEL` | `INFO` | Log level. |
