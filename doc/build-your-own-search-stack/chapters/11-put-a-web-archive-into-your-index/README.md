# 11. Put a web archive into your index

> "Can I supply my index from web archives?"

A web archive holds pages as they were on the day someone saved them. Yours
grows in `warcs/` as you crawl; others arrive as an export from a colleague, a
collection you download, or an archive an institution keeps on the web.

`webarchivescrape` asks an archive which pages it holds and sends each one to
the crawl, so archived words reach your index, including words of pages the live
web no longer has.

## The change

`pywb` serves an archive of your own. It reads the WARC files under `warcs/` and
looks for new ones every thirty seconds, so an archive someone gives you is
searchable soon after you copy it in, with no restart. Smokescreen now allows
the address of the archive, so the crawl is able to fetch a replayed page.

## Try it

Copy WARC files into `warcs/`, beside `compose.yml`. Name each site you want
from them with its own `-url`.

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

## Everything an archive of your own holds

The index of your archive lists the sites it holds, and that list makes the
urls:

```sh
docker compose exec pywb sh -c \
  "cut -d')' -f1 /webarchive/collections/imported/indexes/autoindex.cdxj" \
  | sort -u | awk -F, '{s=$NF; for(i=NF-1;i>=1;i--) s=s "." $i; print "-url", s}' \
  | xargs docker compose run --rm webarchivescrape \
      -pywb-url http://pywb:8080 -pywb-collection imported
```

Run this after you copy an archive in, and every site in it reaches your index.
The same run also reads the WARC files your own crawls wrote, because they are
in `warcs/` too. If you lose your index, this builds it again from your disk,
with no request to the live web.

> [!WARNING]
> The crawl reads a replayed page through the recorder, as it reads any other
> page. So `warcs/` holds those pages a second time, under the address of the
> archive, and the command above lists that address as a site. Remove it from
> the list, or a second run crawls pages your index already has.

## An archive you do not host

`-pywb-url` can name an archive that is already on the web, with the collection
that archive publishes. Nothing is copied to your disk, and the crawl reads the
pages from there. Keep `-page-limit` low: the archive belongs to someone else,
and every page you select is a request they serve.

## Where you are now

You have a search stack that indexes the live web, your own archive of it, and
any archive you are given or allowed to ask.

The rest is tuning. `side-roads/` holds the alternatives worth knowing about,
and every `configuration.md` under `services/` documents settings this journey
left at their defaults.
