# Build your own search stack

Each chapter directory contains a complete, independent Docker Compose stack.
Use the chapters in order to learn the stack, or start with the capability you
need.

## Prepare a chapter

Install Docker Engine with the Docker Compose plugin. Run `docker compose down`
in the previous chapter directory because the chapters publish the same ports.

From the chapter directory:

```sh
cp .env.example .env
```

Complete every empty value in `.env`. Set `YACY_ADVERTISE_HOST` to the address
that YaCy peers use to reach port 8090. Generate each secret with
`openssl rand -hex 32`.

Docker Compose creates separate volumes for each chapter. Data does not move
between chapters.

## Chapters

| # | Capability |
| --- | --- |
| 1 | [Join the YaCy network with one peer](chapters/01-join-the-yacy-network-with-one-peer) |
| 2 | [Give your peer a crawler](chapters/02-give-your-peer-a-crawler) |
| 3 | [Search your index from a browser](chapters/03-search-your-index-from-a-browser) |
| 4 | [Crawl every search result you open](chapters/04-crawl-every-search-result-you-open) |
| 5 | [Index pages that JavaScript builds](chapters/05-index-pages-that-javascript-builds) |
| 6 | [Share one fetch between three readers](chapters/06-share-one-fetch-between-three-readers) |
| 7 | [Put a web archive into your index](chapters/07-put-a-web-archive-into-your-index) |
| 8 | [Keep every page as a Web ARChive file](chapters/08-keep-every-page-as-a-warc-file) |
| 9 | [Let an AI assistant use your web](chapters/09-let-an-ai-assistant-search-and-read-your-web) |
| 10 | [Collect search service metrics](chapters/10-collect-metrics-from-every-service) |
| 11 | [Watch the peer and crawler on dashboards](chapters/11-watch-the-stack-on-dashboards) |

## Alternatives

[Use Elasticsearch instead of Manticore](side-roads/elasticsearch) with chapter
10 or 11.
