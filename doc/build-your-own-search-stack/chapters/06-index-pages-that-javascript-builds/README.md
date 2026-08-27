# 6. Index pages that JavaScript builds

> "I search for a phrase I know is on a page I crawled last week. No result."

Look at what your crawler received. Many sites send an empty shell and a bundle
of JavaScript that builds the page in the browser. The crawler reads the shell,
finds a navigation menu and a spinner, and indexes exactly that.

So this chapter puts a browser in the fetch path, which takes two services.
`lightpanda` is the browser itself. `renderproxy` is what your fetchers talk
to: it takes the same fetch they already make, has the browser load the page
and run its scripts, and returns what the browser ended up with. `yacycrawler`,
`corpustext`, and the peer all fetch through it now, rather than straight
through Smokescreen, which still guards the egress — the browser reaches the
internet through it.

`renderproxy` gives the browser the address of Smokescreen with each page it
sends it to load. The browser's own proxy setting points at a port where
nothing listens, so a page that `renderproxy` did not send fails instead of
going out unguarded.

Lightpanda is built for this work rather than being a Chromium with the window
switched off, which is why the whole stack still fits on a small machine. It
covers ordinary script-built pages, and `renderproxy` drives any browser that
speaks CDP, so you can point it at Chrome for a site that needs one.

## The change

Three services stop fetching pages directly. `corpustext` and the peer fetch
because a scrape request asked them to:

```yaml
      SCRAPE_PROXY_URL: http://renderproxy:8080
      SCRAPE_PROXY_DIAL_MODE: absolute-url
```

`yacycrawler` fetches for its own crawl, and names the same setting after itself:

```yaml
      YACYCRAWLER_FETCH_PROXY_URL: http://renderproxy:8080
      YACYCRAWLER_FETCH_PROXY_DIAL_MODE: absolute-url
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
