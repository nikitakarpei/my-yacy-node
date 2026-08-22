# firecrawlshim configuration

The service is configured entirely through environment variables.

| Variable | Default | Meaning |
|---|---|---|
| `CRAWL_NATS_URL` | required | NATS URL of the broker carrying the crawl orders stream. |
| `NATS_ORDERS_SUBJECT` | `yacy.crawl.orders` | Subject each crawl order is placed on. |
| `FIRECRAWLSHIM_CRAWL_OUTCOMES_TARGET` | required | gRPC target of the crawler that answers what crawling did with a URL. |
| `FIRECRAWLSHIM_MARKDOWN_CORPUS_TARGET` | required | gRPC target of the corpus that holds the markdown of a page. |
| `FIRECRAWLSHIM_LISTEN_ADDR` | `:8093` | Address serving the Firecrawl scrape API. |
| `FIRECRAWLSHIM_RECALL_LIMIT` | `30s` | Time a scrape waits for the corpus before it reports the markdown unavailable. |
| `FIRECRAWLSHIM_POLL_INTERVAL` | `500ms` | Time between two reads of the corpus while a scrape waits. |
| `FIRECRAWLSHIM_MAX_IN_FLIGHT` | `256` | Number of scrapes that can wait at the same time. |
