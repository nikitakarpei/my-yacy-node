# impersonateproxy configuration

You configure impersonateproxy through environment variables.

## Proxy

| Variable | Default | Meaning |
|---|---|---|
| `IMPERSONATEPROXY_LISTEN_ADDR` | `:8080` | Address where the forward proxy accepts requests. |
| `EGRESS_PROXY_URL` | required | HTTP proxy that page requests leave through, in a `CONNECT` tunnel. |
| `IMPERSONATEPROXY_FETCH_TIMEOUT` | `30s` | Longest time for one page, from the tunnel to the last byte of the body. |
| `IMPERSONATEPROXY_DEFAULT_REFERER` | empty | `Referer` that the proxy sends when the client sends none. Empty sends no `Referer`. |

## Operations

| Variable | Default | Meaning |
|---|---|---|
| `IMPERSONATEPROXY_OPS_ADDR` | `:9090` | Address that serves `/metrics`. |
| `LOG_LEVEL` | `INFO` | Log level. |
