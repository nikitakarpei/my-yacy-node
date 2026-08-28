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

## Enabling the plugin

Add the plugin's directory to SearXNG's Python path, then add it to `settings.yml`:

```yaml
plugins:
  result_link_router.SXNGPlugin:
    active: true
```
