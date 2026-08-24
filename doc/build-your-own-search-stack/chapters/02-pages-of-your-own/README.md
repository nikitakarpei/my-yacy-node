# 2. Pages of your own

> "How do I put pages I care about into it?"

The node does not crawl, on purpose: crawling is bursty and memory-hungry, and
the peer is meant to survive on small hardware. So you add a crawler and a queue
between the two. You give `yacycrawler` a starting URL and a profile. It fetches
what the profile admits and publishes a scrape request naming every page it
reached. The node reads those requests, fetches each page itself, and stores the
words as postings — the same postings it already trades with the network.

Orders arrive as JSON on a NATS subject. There is no interface for writing one
yet, so this chapter uses `crawl-console`, a NATS shell you start on demand, to
publish one by hand. That is a gap in the stack, not a workflow, and a later
chapter replaces it with a link you click.

## The change

`yacy-rwi-node` gains two settings. Without them it ignores the scrape requests:

```yaml
      SCRAPE_REQUEST_NATS_URL: nats://nats:4222
      SCRAPE_PROXY_URL: http://smokescreen:4750
```

## Try it

Seed URLs must already be canonical — a bare domain needs its trailing slash:

```sh
docker compose run --rm crawl-console pub yacy.crawl.orders \
  '{"OrderID":"first","SeedURLs":["https://example.org/"],
    "Profile":{"Name":"first","Scope":1,"MaxDepth":1,
    "URLMustMatch":".*","MaxPagesPerHost":50}}'

docker compose logs -f yacycrawler yacy-rwi-node
```

`Scope` is `0` wide, `1` domain, `2` subpath.

## Where you are now

Your peer holds pages you chose and offers them to the network. You still cannot
search them: a posting answers another peer's DHT query, not a person's.

> "Can I search my own pages, from this machine, by word?"
