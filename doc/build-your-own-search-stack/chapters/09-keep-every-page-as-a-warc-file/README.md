# 9. Keep every page as a Web ARChive file

> "Can I keep the pages that I crawl as a web archive?"

An index keeps searchable text and cannot reproduce a page. Recording each
fetched page gives you a portable copy for replay, transfer, or later indexing.

## What this chapter adds

- `warcprox` saves every fetched page as a Web ARChive (WARC) file in `warcs/`.

The stack does not prune this directory. Monitor its size and stop crawling
before the disk fills.

## Start

Start the stack:

```sh
docker compose up -d
```

## Use

Submit a crawl with the [chapter 3 command](../03-give-your-peer-a-crawler#use),
then list the recorded files:

```sh
ls -lh warcs/
```

A file with an `.open` suffix is still being written. Use only closed files for
replay or transfer. Script-built pages can replay with less text because the
WARC file holds the response from the origin.

## More information

- [Index saved WARC files](../07-put-a-web-archive-into-your-index)

List recorder options:

```sh
docker compose exec warcprox warcprox --help
```
