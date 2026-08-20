# searxng-result-router configuration

The plugin is configured through environment variables in the SearXNG process, and enabled
through SearXNG's own `settings.yml`.

Rewritten links point at `/visit` on the origin that serves the results page, so a reverse
proxy in front of SearXNG must send `/visit` to `visitcrawl`.

## Environment

| Variable | Default | Meaning |
|---|---|---|
| `VISITCRAWL_LINK_SECRET` | required | Secret the plugin signs rewritten links with. Set it to the same value as the `visitcrawl` that serves `/visit`. |
| `RESULT_LINK_ROUTER_LINK_LIFETIME` | `86400` | Seconds a rewritten link stays valid after the plugin issues it. |
| `RESULT_LINK_ROUTER_DISABLE_HEADER` | `X-Result-Link-Router-Disable` | Name of the request header that turns off link rewriting for a single request. |

## Disabling rewriting for a request

A request carrying the header named by `RESULT_LINK_ROUTER_DISABLE_HEADER` (any value) skips
rewriting; its results link straight to their destinations, same as when the plugin can't
rewrite a link.

## Enabling the plugin

Add the plugin's directory to SearXNG's Python path, then add it to `settings.yml`:

```yaml
plugins:
  result_link_router.SXNGPlugin:
    active: true
```
