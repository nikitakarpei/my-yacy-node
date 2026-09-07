# Build your own search stack

This guide builds a self-hosted web search stack in small steps. The stack
joins the YaCy peer-to-peer search network, provides a search page, crawls web
pages, and keeps a local full-text index. Later chapters add browser
rendering, web archives, AI access, metrics, and dashboards.

Start with chapter 1 and continue in order. Each chapter provides a complete,
independent Docker Compose stack with its own volumes and data. A decimal
chapter, such as 1.1, is an optional alternative to the chapter before it.

## Prepare a chapter

Install Docker Engine with the Docker Compose plugin. Run `docker compose down`
in the previous chapter directory because the chapters publish the same ports.

From the chapter directory:

```sh
cp .env.example .env
```

Complete every empty value in `.env`.

## Chapters

| # | Capability |
| --- | --- |
| 1 | [Join the YaCy network with one peer](chapters/01-join-the-yacy-network-with-one-peer) |
| 1.1 | [Join the network from behind NAT](chapters/01.1-join-the-network-from-behind-nat) |
| 2 | [Search the YaCy network from your browser](chapters/02-search-the-yacy-network-from-your-browser) |
| 3 | [Give your peer a crawler](chapters/03-give-your-peer-a-crawler) |
| 4 | [Search your index from a browser](chapters/04-search-your-index-from-a-browser) |
| 4.1 | [Use Elasticsearch instead of Manticore](chapters/04.1-use-elasticsearch-instead-of-manticore) |
| 5 | [Crawl every search result you open](chapters/05-crawl-every-search-result-you-open) |
| 6 | [Index pages that JavaScript builds](chapters/06-index-pages-that-javascript-builds) |
| 7 | [Share one fetch between three readers](chapters/07-share-one-fetch-between-three-readers) |
| 8 | [Put a web archive into your index](chapters/08-put-a-web-archive-into-your-index) |
| 9 | [Keep every page as a Web ARChive file](chapters/09-keep-every-page-as-a-warc-file) |
| 10 | [Let an AI assistant use your web](chapters/10-let-an-ai-assistant-search-and-read-your-web) |
| 11 | [Collect search service metrics](chapters/11-collect-metrics-from-every-service) |
| 12 | [Watch search services on dashboards](chapters/12-watch-the-stack-on-dashboards) |
