# 3. Search your index from a browser

> "Can I search my own pages, from this machine, by word?"

Postings are built for the DHT: they answer "which peer knows this word" rather
than "show me the page". For your own searching you want a full-text index, so
this chapter adds a second reader of the same scrape requests.

`corpustext` reads every scrape request, fetches the page, and writes its text
into `manticore`. `searxng` gives you the page to type into. It queries
Manticore through the `crawled_text_search` engine, alongside the web engines it
ships with, and merges both into one list. Your crawled pages and the live web
sit in the same results.

Manticore is here because it indexes a few hundred thousand pages inside a few
hundred megabytes. Elasticsearch does the same job with more memory; see
`side-roads/elasticsearch` if you already run one.

## The change

```sh
printf 'SEARXNG_SECRET=%s\n' "$(openssl rand -hex 32)" >> .env
```

`CORPUSTEXT_LANGUAGES` decides which per-language indices exist. `en` is the
default; add more as a comma-separated list.

## Try it

```sh
docker compose up -d
```

Open `http://localhost:8080` and search for a phrase from a page you crawled in
the previous chapter. Prefix a query with `!ct` to see the local index alone.

Nothing came back? `corpustext` only indexes pages requested after it started.
Order that crawl again — the crawler skips a URL it fetched within the last
hour, so give it `YACYCRAWLER_RECRAWL_GRACE=0` or wait.

## Where you are now

You have a search page over your own crawl. Growing that crawl still means
writing JSON by hand.
