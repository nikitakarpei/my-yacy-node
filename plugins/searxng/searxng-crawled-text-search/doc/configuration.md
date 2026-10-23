# searxng-crawled-text-search configuration

The engine is configured entirely through its `engines:` entry in SearXNG's own `settings.yml` —
no environment variables.

## Enabling the engine

Add the engine's directory to SearXNG's `engines/` folder (or mount `crawled_text_search.py`
there), then add it to `settings.yml`:

```yaml
engines:
  - name: crawled text search
    engine: crawled_text_search
    shortcut: ct
    categories: general
    enable_http: true
    corpustext_url: http://corpustext:8096
```

| Key | Default | Meaning |
|---|---|---|
| `corpustext_url` | required | Base URL of the `corpustext` search. |

The engine does not know which search index `corpustext` writes to. To change the search
index, change the configuration of `corpustext` only.
