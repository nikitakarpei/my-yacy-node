# 3. Give your peer a crawler

> "How do I put pages I care about into the network?"

The peer stores postings that reach it from the network. A crawler lets you
choose pages and contribute their words to the shared index.

## What this chapter adds

- `yacycrawler` visits the pages you name and follows links within the limits
  you set.
- `pagescrape` reads each requested page once and offers it to the indexes.
- `scrape-request-bridge` makes a scrape request from each crawled page that
  allows indexing.
- `nats` keeps unfinished crawl and indexing work on disk, so services can
  continue after a restart.
- `crawl-console` lets you submit a crawl from the command line.

## Start

Start the stack:

```sh
docker compose up -d
```

## Use

Submit a crawl for `example.org`:

```sh
docker compose run --rm crawl-console pub yacy.crawl.orders \
  '{"OrderID":"first","SeedURLs":["https://example.org/"],
    "Profile":{"Name":"first","Scope":1,"MaxDepth":1,
    "URLMustMatch":".*","MaxPagesPerHost":50}}'

curl -fsS localhost:9090/metrics \
  | grep 'vault_collection_entries{collection="rwi"}'
```

A nonzero value confirms that the node stored reverse word index postings from
the crawl.

## More information

- [Crawler configuration](../../../../services/yacycrawler/doc/configuration.md)
- [Crawl behavior](../../../../services/yacycrawler/doc/specification.md)
- [Node configuration](../../../../services/yacynode/doc/configuration.md)
