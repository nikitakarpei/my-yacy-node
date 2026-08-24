# 4. Pages you open

> "What if the pages I open were crawled for me?"

Searching is already a statement about what interests you, and until now the
stack threw it away. This chapter puts one service in the middle of it.
`visitcrawl` sits between the results and the pages they point at: every result
link now goes to it first, and it places a crawl order for that page before
sending your browser on. The page you read today is in your index tomorrow.

It answers at `/visit`, and such a link works only from the address the results
came from, so `caddy` goes in front: the search page and `/visit` now share one
port, the `http://localhost:8080` you have been searching at all along.

## The change

```sh
printf 'VISITCRAWL_LINK_SECRET=%s\n' "$(openssl rand -hex 32)" >> .env
```

`searxng` gives up its published port to `caddy`.

## Try it

```sh
docker compose up -d
```

Search, open any result, then watch the order arrive:

```sh
docker compose logs -f visitcrawl yacycrawler
```

Search for something from that page a minute later.

## Where you are now

Your index grows while you use it. Each visit brings the page you opened, and
one link-step past it.

> "How far past that page can I let it go?"
