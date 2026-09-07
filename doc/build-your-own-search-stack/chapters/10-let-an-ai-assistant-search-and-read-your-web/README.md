# 10. Let an AI assistant use your web

> "Can my AI assistant use my search stack?"

An assistant's built-in web tools cannot see your local index or stored page
text. This chapter gives the assistant access to both.

## What this chapter adds

- `corpusmarkdown` stores the latest crawled page text as Markdown.
- `webresearchmcp` lets your assistant search SearXNG and read page text.

## Start

Start the stack:

```sh
docker compose up -d
```

## Use

Connect a Model Context Protocol client that supports HTTP to
`http://localhost:8095/mcp`. For Claude Code:

```sh
claude mcp add --transport http my-stack http://localhost:8095/mcp
```

Ask the client to search and read a result. Search for text from that page at
`http://localhost:8080` to confirm that the read added it to the index.

## More information

- [Tool configuration](../../../../services/webresearchmcp/doc/configuration.md)
- [Tool inputs and results](../../../../services/webresearchmcp/doc/specification.md)
- [Markdown corpus configuration](../../../../services/corpusmarkdown/doc/configuration.md)
