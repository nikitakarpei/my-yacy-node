# 10. Put a web archive into your index

> "Can I supply my index from web archives?"

A web archive holds pages as they were on the day someone saved them. One
reaches you as an export from a colleague or as a collection you download.

`webarchivescrape` asks an archive which pages it holds and sends each one to
the crawl, so archived words reach your index, including words of pages the live
web no longer has.

## The change

`pywb` serves an archive of your own. It reads the WARC files under `imported/`
and looks for new ones by itself, so an archive someone gives you is searchable
soon after you copy it in, with no restart.

`pywb.yaml` turns off frame replay. Keep it: without it the crawl indexes
pywb's frame and not the archived page.

The archive sits on a network of its own that has no route out. Smokescreen
reaches it and now allows its address, so the crawl is able to fetch a replayed
page, and the archive itself is able to fetch nothing.

## Try it

Copy WARC files into `imported/`, beside `compose.yml`, and wait thirty seconds
for pywb to find them. Name each site you want from them with its own `-url`.

```sh
docker compose up -d

docker compose run --rm webarchivescrape \
  -pywb-url http://pywb:8080 -pywb-collection imported \
  -url example.com -url example.org -page-limit 100
```

The command prints each page as it sends the scrape request for it, so a run
that stops early tells you what reached the crawl. Search for a word from those
pages at `http://localhost:8080`, and they are in the results. Open
`http://localhost:8081` to browse the archive yourself.

`webarchivescrape -h` states the rest: match types, capture date bounds, and
`-dry-run`, which lists the pages and sends none. An archive keeps every capture
of a page, and the command selects the newest, so one run publishes a page once.

## Everything an archive holds

The index of your archive lists the sites it holds, and that list makes the
urls:

```sh
docker compose exec pywb sh -c \
  "cut -d')' -f1 /webarchive/collections/imported/indexes/autoindex.cdxj" \
  | sort -u | awk -F, '{s=$NF; for(i=NF-1;i>=1;i--) s=s "." $i; print "-url", s}' \
  | xargs docker compose run --rm webarchivescrape \
      -pywb-url http://pywb:8080 -pywb-collection imported
```

Run this after you copy an archive in, and every site in it reaches your index,
whether or not you know which sites the files hold.

## Where you are now

You can search an archive next to the live web, and the pages of a site that is
gone are as easy to find as the pages of one that is still up.

What you keep of your own crawls is still only words. Every page this stack
fetches goes through it once, and nothing writes it down.
