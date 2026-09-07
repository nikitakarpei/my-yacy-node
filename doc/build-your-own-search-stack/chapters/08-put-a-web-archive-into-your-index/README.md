# 8. Put a web archive into your index

> "Can archived pages become part of my search index?"

An archive can preserve pages that changed or disappeared. Importing its pages
makes that saved content searchable with the live crawl.

## What this chapter adds

- `pywb` lets you browse the Web ARChive files that you supply.
- `webarchivescrape` is a command that lets you select pages from those files
  and add them to your search index.

## Start

Create the archive directory and copy Web ARChive (WARC) files into it:

```sh
mkdir -p imported
```

Start the stack:

```sh
docker compose up -d
```

## Use

Submit up to 100 archived pages from each named domain:

```sh
docker compose run --rm webarchivescrape \
  -pywb-url http://pywb:8080 -pywb-collection imported \
  -url example.com -url example.org -page-limit 100
```

Search the indexed pages at `http://localhost:8080`. Browse the archive at
`http://localhost:8081`.

## More information

List archive selection and dry-run options:

```sh
docker compose run --rm webarchivescrape -h
```
