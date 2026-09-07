# 4. Search your index from a browser

> "Can I search my own crawl from a browser?"

This chapter puts text from your crawl in a local full-text index and adds its
results to your browser search page.

## What this chapter adds

- `corpustext` puts text from crawled pages into your local search index.
- `manticore` keeps the local full-text index.
- The `searxng-crawled-text-search` plugin adds the local index to the search
  page.

## Start

Start the stack before you submit a crawl:

```sh
docker compose up -d
```

## Use

Submit a crawl with the [chapter 3 command](../03-give-your-peer-a-crawler#use).
Open `http://localhost:8080` and search for text from a crawled page. Prefix the
query with `!ct` to show only the local index.

## More information

- [Full-text indexer configuration](../../../../services/corpustext/doc/configuration.md)
- [SearXNG local search configuration](../../../../plugins/searxng/searxng-crawled-text-search/doc/configuration.md)
- [Chapter 4.1: Use Elasticsearch instead of Manticore](../04.1-use-elasticsearch-instead-of-manticore)
