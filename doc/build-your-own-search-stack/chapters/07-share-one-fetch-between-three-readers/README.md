# 7. Share one fetch between three readers

> "Why does the stack fetch one page several times?"

`yacycrawler` fetches a page to discover links. `pagescrape` reads each
requested page once and offers the same content to `corpustext` and
`yacy-rwi-node`. A shared cache lets link discovery and the scrape reuse one
fetch from the origin.

## What this chapter adds

- `squid` keeps a temporary copy of each rendered page, so services can share
  one fetch.

The cache uses memory and disk. Review its limits before a large crawl.

## Start

Start the stack:

```sh
docker compose up -d
```

## Use

Submit a crawl with the [chapter 3 command](../03-give-your-peer-a-crawler#use),
then inspect the cache results:

```sh
docker compose logs -f squid
```

Repeated requests for a page report `TCP_MEM_HIT` or `TCP_CF_HIT`.

## More information

- [Cache configuration](../../building-blocks/squid/squid.conf)
- [Rendering configuration](../../../../services/renderproxy/doc/configuration.md)
