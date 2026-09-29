# spamproxy configuration

You configure spamproxy through environment variables.

## Proxy

| Variable | Default | Meaning |
|---|---|---|
| `SPAMPROXY_LISTEN_ADDR` | `:8080` | Address where the forward proxy accepts requests. |
| `EGRESS_PROXY_URL` | required | HTTP proxy that page requests leave through. |
| `SPAMPROXY_EGRESS_PROXY_DIAL_MODE` | `tunnel` | How page requests address that proxy: `tunnel` opens a CONNECT tunnel for `https`, `absolute-url` names the whole address in the request line. |
| `SPAMPROXY_RESPONSE_HEADER_TIMEOUT` | `10s` | Longest time from a request until the proxy sends the response headers. A shorter `Prefer: wait` replaces it. |
| `SPAMPROXY_RELAY_IDLE_TIMEOUT` | `30s` | Longest wait for the next bytes of a body that the proxy relays after the verdict. |

## Model

| Variable | Default | Meaning |
|---|---|---|
| `SPAMPROXY_MODEL_PATH` | required | Model file. The service does not start without a usable model file. |

## Limits

| Variable | Default | Meaning |
|---|---|---|
| `SPAMPROXY_PAGE_BYTE_CEILING` | `1048576` | Most bytes of a page that the proxy reads and decodes to assess it. |
| `SPAMPROXY_MAX_PAGES_READ_AT_ONCE` | `64` | Pages that the proxy reads and assesses at the same time. A page past this limit waits. It gets `503` when its response headers are due before a place is free. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `SPAMPROXY_OPS_ADDR` | `:9090` | Address that serves `/metrics`. |
| `LOG_LEVEL` | `INFO` | Log level. |
