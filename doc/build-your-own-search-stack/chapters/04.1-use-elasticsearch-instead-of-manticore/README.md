# 4.1. Use Elasticsearch instead of Manticore

> "Can I keep my local search index in Elasticsearch?"

This chapter stores crawled text in Elasticsearch instead of Manticore. Use it
when you already operate Elasticsearch or need its tools.

## What this chapter changes

- `elasticsearch` stores and searches the local full-text index.
- `corpustext` writes crawled text to Elasticsearch.
- `searxng` reads local search results from Elasticsearch.

This chapter does not move documents from Manticore. Submit a new crawl to fill
the Elasticsearch index.

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
- [SearXNG local search configuration](../../../../plugins/searxng/searxng-crawled-text-search/doc/configuration.md)
