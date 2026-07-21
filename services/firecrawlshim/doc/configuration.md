# firecrawlshim configuration

The service is configured entirely through environment variables.

| Variable | Default | Meaning |
|---|---|---|
| `FIRECRAWLSHIM_RECALL_TARGET` | required | gRPC target of the corpusrecall service to recall representations from. |
| `FIRECRAWLSHIM_LISTEN_ADDR` | `:8093` | Address serving the Firecrawl scrape API. |
| `FIRECRAWLSHIM_RECALL_TIMEOUT` | `30s` | Time a scrape waits on corpusrecall before returning a gateway error. |
