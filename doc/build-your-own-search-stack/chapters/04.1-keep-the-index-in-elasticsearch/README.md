# 4.1. Keep the index in Elasticsearch

> "Can I keep my local search index in Elasticsearch?"

This chapter keeps crawled text in Elasticsearch instead of the built-in index.
Use it when you already run Elasticsearch, need its tools, or run more than one
`corpustext`.

## What this chapter changes

- `elasticsearch` stores and searches the local full-text index.
- `corpustext` writes crawled text to Elasticsearch and searches it.
- `searxng` gets local search results from `corpustext`, as in chapter 4.

Pages in the chapter 4 index do not move. Submit a new crawl to fill
Elasticsearch.

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

- [Elasticsearch index operation](../../../../services/corpustext/doc/elasticsearch.md)
- [Full-text indexer configuration](../../../../services/corpustext/doc/configuration.md)
- [SearXNG local search configuration](../../building-blocks/searxng/settings-local-index-and-yacy-network.yml)
