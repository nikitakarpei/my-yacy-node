# 4.2. Keep the index in Manticore

> "Can I keep my local search index in Manticore?"

This chapter keeps crawled text in Manticore instead of the built-in index. Use
it when you already run Manticore or run more than one `corpustext`.

## What this chapter changes

- `manticore` stores and searches the local full-text index.
- `corpustext` writes crawled text to Manticore and searches it.
- `searxng` gets local search results from `corpustext`, as in chapter 4.

Pages in the chapter 4 index do not move. Submit a new crawl to fill
Manticore.

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

- [Manticore index operation](../../../../services/corpustext/doc/manticore.md)
- [Full-text indexer configuration](../../../../services/corpustext/doc/configuration.md)
- [SearXNG local search configuration](../../../../plugins/searxng/searxng-crawled-text-search/doc/configuration.md)
