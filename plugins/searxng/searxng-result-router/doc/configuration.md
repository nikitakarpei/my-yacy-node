# searxng-result-router configuration

The plugin is configured through an environment variable in the SearXNG process, and enabled
through SearXNG's own `settings.yml`.

## Environment

| Variable | Default | Meaning |
|---|---|---|
| `YACYVISITCRAWL_BASE_URL` | required | Base URL of the `yacyvisitcrawl` that rewritten result links route through, e.g. `http://yacyvisitcrawl:8091`. |
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
