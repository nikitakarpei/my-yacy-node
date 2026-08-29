# 6. Share one fetch between three readers

> "Why does the stack fetch one page several times?"

`yacycrawler` fetches a page to discover links. `corpustext` and
`yacy-rwi-node` fetch the page to build their different indexes. Each service
owns one job and works independently, so the same page is fetched three times.
A shared cache lets them reuse one fetch.

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

Submit a crawl with the [chapter 2 command](../02-give-your-peer-a-crawler#use),
then inspect the cache results:

```sh
docker compose logs -f squid
```

Repeated requests for a page report `TCP_MEM_HIT` or `TCP_CF_HIT`.

## More information

- [Cache configuration](../../building-blocks/squid/squid.conf)
- [Rendering configuration](../../../../services/renderproxy/doc/configuration.md)
