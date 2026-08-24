# 4. Pages you open

> "What if the pages I open were crawled for me?"

Searching is already a statement about what interests you, and until now the
stack threw it away. This chapter puts a redirect in the middle: every result
link now points at `/visit`, where `visitcrawl` places a crawl order for that
page and sends your browser on to it. The page you read today is in your index tomorrow.

Caddy arrives with it, because a visit link only works from the same address the
results came from. It puts the search page and the visit links on one port. The
peer keeps its own, as before.

## The change

```sh
printf 'VISITCRAWL_LINK_SECRET=%s\n' "$(openssl rand -hex 32)" >> .env
```

`searxng` gives up its published port to `caddy`. You still open
`http://localhost:8080`; nothing else about how you reach the stack changes.

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
