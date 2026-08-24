# 5. How far a crawl goes

> "How far past that page can I let it go?"

No new services here. Every order `visitcrawl` places carries the same profile,
built from environment variables, and this chapter is about three of them.

`VISITCRAWL_MAX_DEPTH` is link-steps from the page you opened: `1` takes the
page and what it links to. `VISITCRAWL_SCOPE` decides which of those links count
— `subpath` stays under the directory you visited, `domain` allows the whole
host, `wide` follows anywhere. `VISITCRAWL_MAX_PAGES_PER_HOST` is the stop, and
it matters most with `wide`, where a single visit can otherwise reach a large
part of the web.

Depth is multiplicative, so raise it slowly. Depth 2 on a well-linked site is
thousands of pages, each one fetched, indexed, and stored on your disk. The
crawler stops a run at `YACYCRAWLER_RUN_PAGE_BUDGET` pages, 1000 by default,
which is a backstop rather than a plan.

## The change

```sh
VISITCRAWL_SCOPE=domain
VISITCRAWL_MAX_DEPTH=1
VISITCRAWL_MAX_PAGES_PER_HOST=100
```

A hand-written order carries its own profile and ignores these, so the console
is still how you crawl one site deeply without changing what every visit does.

## Try it

```sh
docker compose up -d visitcrawl
```

Open a result and watch the pages arrive:

```sh
docker compose logs -f yacycrawler
```

Watch your disk as well: `docker system df -v | grep yacy-data`.

## Where you are now

You decide how much of a site a visit brings in. Some of what arrives is empty.

> "Why do some pages come back with nothing in them?"
