# 7. Share one fetch between three readers

> "Why is the same page fetched three times?"

Three services fetch every page. `yacycrawler` fetches it to follow its links,
then publishes a scrape request naming it. `corpustext` reads that request and
keeps the page text; the peer reads it and records a posting for every word the
page holds. Each of the three fetches the page for itself, so the browser
renders it three times and the origin sees three visits.

`squid` is a caching proxy in front of `renderproxy`. The first request renders
the page; the others read what is stored. When the three arrive together, one
goes on and the other two wait for it.

## The change

The three services point their fetch proxy at the cache. `corpustext` and the
peer:

```yaml
      SCRAPE_PROXY_URL: http://squid:3128
```

`yacycrawler`:

```yaml
      YACYCRAWLER_FETCH_PROXY_URL: http://squid:3128
```

How long a page stays good is the origin's decision. `renderproxy` passes on the
`Cache-Control` and `Last-Modified` it received, and the cache obeys them.
`building-blocks/squid/squid.conf` holds the rest: 64 MB in memory and 1 GB on
disk. The cache accepts every client, which is safe only because it has no
published port.

## Try it

```sh
docker compose up -d

docker compose run --rm crawl-console pub yacy.crawl.orders \
  '{"OrderID":"cached","SeedURLs":["https://example.org/"],
    "Profile":{"Name":"cached","Scope":1,"MaxDepth":1,
    "URLMustMatch":".*","MaxPagesPerHost":50}}'

docker compose logs -f squid
```

Every line ends with what the cache did. `TCP_MISS` is a page the renderer
fetched. `TCP_MEM_HIT` is a page it did not. `TCP_CF_HIT` is a service that
waited for another service's fetch instead of starting its own.

## Where you are now

One page, one visit to the origin, however many services want it.

Every page in your index is one the live web served you, so a page that is gone
is out of your reach, even when somebody kept a copy of it.
