# 3. Search your index from a browser

> "Can I search my own crawl from a browser?"

The peer serves the shared reverse word index to other YaCy peers. A full-text
index and search page make your crawl searchable on your machine.

## What this chapter adds

- `corpustext` puts text from crawled pages into your local search index.
- `manticore` keeps the local full-text index.
- `searxng` shows local and live web results on one search page.
- The `searxng-crawled-text-search` plugin lets SearXNG search that index.

## Start

Start the stack before you submit a crawl:

```sh
docker compose up -d
```

## Use

Submit a crawl with the [chapter 2 command](../02-give-your-peer-a-crawler#use).
Open `http://localhost:8080` and search for text from a crawled page. Prefix the
query with `!ct` to show only the local index.

## More information

- [Full-text indexer configuration](../../../../services/corpustext/doc/configuration.md)
- [SearXNG local search configuration](../../../../plugins/searxng/searxng-crawled-text-search/doc/configuration.md)
- [Chapter 3.1: Use Elasticsearch instead of Manticore](../03.1-use-elasticsearch-instead-of-manticore)
