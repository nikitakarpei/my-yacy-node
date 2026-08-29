# 10. Let an AI assistant search and read your web

> "Can my AI assistant use my stack instead of somebody else's?"

An AI assistant that speaks the Model Context Protocol reaches the web through
whichever search and fetch service its vendor wired in, which leaves your index
and your egress out of its reach.

`webresearchmcp` serves two tools at one endpoint any such AI assistant can be
pointed at. `search_web` answers with the results of your own SearXNG.
`read_page` answers with the markdown of one page, and `corpusmarkdown`, which
arrives with it, is where that markdown is kept.

## The change

Two services join the stack. `corpusmarkdown` reads the scrape requests already
on the stream and fetches each page through `squid`, like the readers of chapter
7, so a page it stores costs the origin nothing more. `webresearchmcp` reaches
SearXNG, that corpus and the broker, keeps nothing of its own, and publishes its
endpoint on `127.0.0.1:8095`.

The scrape a page call asks for is the same request the rest of the stack
listens to, so a page your AI assistant reads goes into your index and, from
chapter 9, into a WARC file.

## Try it

```sh
docker compose up -d
```

Point an AI assistant at `http://localhost:8095/mcp`. In Claude Code:

```sh
claude mcp add --transport http my-stack http://localhost:8095/mcp
```

Ask it to search for something, then to read one of the results. Now search at
`http://localhost:8080` for a word from the page it read, and that page comes
back among the results: reading it put it in your index.

## Where you are now

Your AI assistant searches your index, crawls through your rules, leaves what it
reads on your disk, and tells nobody outside what you asked.

A dozen containers now fetch, index and answer, and none of them tells you how
it is doing.
