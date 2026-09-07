# 5. Crawl every search result you open

> "Can the pages I open enter the crawl automatically?"

Manual crawl orders interrupt normal searching. This chapter starts a crawl
each time you open a search result, so the local index grows while you use it.

## What this chapter adds

- `visitcrawl` is a service that starts a crawl when you open a search result.
  It then sends your browser to the result page.
- The `searxng-result-router` plugin points each SearXNG result at `visitcrawl`.
- `caddy` gives your browser one address for both searching and starting crawls
  from result links.

`visitcrawl` settings limit how far links can lead and how many pages one search
result can add.

## Start

Add crawl limits to `.env`. This example stays under the opened path and stops
at 25 pages per host:

```dotenv
VISITCRAWL_SCOPE=subpath
VISITCRAWL_MAX_DEPTH=1
VISITCRAWL_MAX_PAGES_PER_HOST=25
```

Start the stack:

```sh
docker compose up -d
```

## Use

Search at `http://localhost:8080` and open a result. After the crawl completes,
search for text from that page. If the page does not enter the index, inspect
`visitcrawl` and `yacycrawler` logs.

## More information

- [Visit crawl configuration](../../../../services/visitcrawl/doc/configuration.md)
- [Result router configuration](../../../../plugins/searxng/searxng-result-router/doc/configuration.md)
