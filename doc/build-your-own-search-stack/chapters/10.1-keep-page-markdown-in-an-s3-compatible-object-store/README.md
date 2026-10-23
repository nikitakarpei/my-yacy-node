# 10.1. Keep page Markdown in an S3-compatible object store

> "Can I keep the page Markdown in an object store?"

This chapter keeps page Markdown in an S3-compatible object store instead of a
volume. Use it when you already run an object store, or run `corpusmarkdown` on
more than one host.

## What this chapter changes

- `garage` keeps the Markdown on disk as an S3-compatible object store.
- `corpusmarkdown` writes the Markdown to Garage.

Markdown from the chapter 10 volume does not move. Read a page again to store
it in Garage.

## Start

Start the stack:

```sh
docker compose up -d
```

## Use

Use the stack as in [chapter 10](../10-let-an-ai-assistant-search-and-read-your-web#use).

## More information

- [Markdown corpus configuration](../../../../services/corpusmarkdown/doc/configuration.md)
