# 6. Index pages that JavaScript builds

> "Why is text from a crawled page missing from search?"

Some pages send their text only after JavaScript runs. Browser rendering lets
the indexers read the page that a person sees.

## What this chapter adds

- `lightpanda` is a lightweight Chrome DevTools Protocol-compatible browser that
  loads pages and runs their scripts. You can replace it with Chrome when a site
  needs it.
- `renderproxy` acts as an HTTP proxy for the stack. It loads each requested
  page in a Chrome DevTools Protocol-compatible browser and returns the rendered
  content.

Browser rendering uses more memory and time than a direct page fetch. Set the
number of pages that can render at once to fit the capacity of the host.

## Start

Start the stack:

```sh
docker compose up -d
```

## Use

Submit a crawl with the [chapter 3 command](../03-give-your-peer-a-crawler#use)
for a page that builds its content with JavaScript. Search at
`http://localhost:8080` for text that appears only after the page loads.

Check rendering failures before you increase time or size limits:

```sh
docker compose logs -f renderproxy lightpanda
```

## More information

- [Rendering configuration](../../../../services/renderproxy/doc/configuration.md)
- [Rendering behavior and limits](../../../../services/renderproxy/doc/specification.md)
