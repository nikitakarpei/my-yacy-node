# 6. Nothing comes back

> "I search for a phrase I know is on a page I crawled last week. No result."

Look at what your crawler received. Many sites send an empty shell and a bundle
of JavaScript that builds the page in the browser. The crawler reads the shell,
finds a navigation menu and a spinner, and indexes exactly that.

So this chapter puts a browser in the fetch path. `renderproxy` accepts the same
fetch every service already makes, drives `lightpanda` to load the page, waits
for the scripts to finish, and returns the result. `yacycrawler`, `corpustext`,
and the peer all point their page fetches at it instead of straight at
Smokescreen. Smokescreen still guards the egress: the browser reaches the
internet through it.

Lightpanda is a browser built for this rather than a Chromium with the window
switched off, which is why the whole stack still fits on a small machine. It
covers ordinary script-built pages; `renderproxy` drives any browser that
speaks CDP, so you can point it at Chrome for a site that needs one.

## The change

Three services stop fetching pages directly:

```yaml
      SCRAPE_PROXY_URL: http://renderproxy:8080
      SCRAPE_PROXY_DIAL_MODE: absolute-url
```

`EGRESS_PROXY_URL` on the peer is unchanged: peer-to-peer traffic never renders.

## Try it

```sh
docker compose up -d
```

Crawl a site you know is script-built, then search for a phrase from the middle
of its text. Compare with a page crawled before this chapter — that one keeps
its thin index until something crawls it again.

## Where you are now

The index holds what a reader would have seen. Two services now read every
scrape request, each at its own pace, and a browser sits in the fetch path of
both.

> "What happens if one of them falls behind?"
