# 10. Keeping them

> "Can I save what I crawl in a format anyone can open?"

Your index holds words about pages rather than the pages themselves, so when a
site rewrites something or takes it down, you keep the words and lose the thing
they described.

`warcprox` records what the stack fetches. Every page goes into a WARC file
under `warcs/`, with the status line, the headers and the bytes the origin sent.
WARC is ISO 28500, so those files open in other people's tools, and they will
still open long after this stack is gone.

The recorder sits below the browser, which means what it keeps is what the
origin sent rather than what the browser made of it. Smokescreen still guards
the egress, and it still refuses every private address except the recorder.

## The change

Crawling works as it did before. Smokescreen now hands whatever it allows
through to the recorder, and the stack trusts the certificate authority the
recorder generates for HTTPS, so nothing has to stop checking certificates.

## Try it

```sh
docker compose up -d

docker compose run --rm crawl-console pub yacy.crawl.orders \
  '{"OrderID":"kept","SeedURLs":["https://example.org/"],
    "Profile":{"Name":"kept","Scope":1,"MaxDepth":1,
    "URLMustMatch":".*","MaxPagesPerHost":50}}'

ls warcs/
```

The files land in `warcs/`, beside `compose.yml`. A name ending in `.open` is
one that is still being written, and `warcprox` closes it after five idle
minutes.

Load a closed file into `https://replayweb.page` and the page comes back as it
was on the day you crawled it, with nothing to install.

## What the files give you

* Hand them to someone else, and they load in pywb, OpenWayback or
  replayweb.page.
* Keep what a page said, long after the site has changed it or taken it down.
* Add them to a web archive that is already running, or take a copy of one.

## Disk

`warcs/` grows with every crawl and nothing prunes it. The recorder stores each
set of bytes once and writes a short `revisit` record when it meets the same
bytes again, but that only slows the growth down. Keep an eye on the directory
before you crawl anything large.

## Where you are now

You have the pages themselves, in a format that outlives this stack, rather than
only words about them. Nothing in the stack reads them back yet.

The rest is tuning. `side-roads/` holds the alternatives worth knowing about,
and every `configuration.md` under `services/` documents settings this journey
left at their defaults.
