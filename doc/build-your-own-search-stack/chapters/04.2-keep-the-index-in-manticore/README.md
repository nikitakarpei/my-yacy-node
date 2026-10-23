# 4.2. Keep the index in Manticore

> "Can I keep my local search index in Manticore?"

This chapter stores crawled text in Manticore instead of the index that
`corpustext` keeps itself. Use it when you already operate Manticore or run
more than one `corpustext`.

## What this chapter changes

- `manticore` stores and searches the local full-text index.
- `corpustext` writes crawled text to Manticore and searches it.
- `searxng` gets local search results from `corpustext`, as in chapter 4.

This chapter does not move documents from the index of chapter 4. Submit a new
crawl to fill the Manticore index.

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
