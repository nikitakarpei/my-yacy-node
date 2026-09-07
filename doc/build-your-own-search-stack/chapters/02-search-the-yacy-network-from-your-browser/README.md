# 2. Search the YaCy network from your browser

> "Can my search page include results from the YaCy network?"

The peer supports the YaCy network, but it does not provide a search page.
This chapter searches the shared reverse word index across live YaCy peers.

## What this chapter adds

- `yacydhtsearch` discovers YaCy peers, asks them for each query, and merges
  their results.
- `searxng` provides the browser search page. Its native YaCy engine sends
  general searches to `yacydhtsearch`.

## Start

Start the stack before you search:

```sh
docker compose up -d
```

Give `yacydhtsearch` time to discover reachable peers after its first start.

## Use

Open `http://localhost:8080` and search as usual. Results can come from the
YaCy network and the other enabled SearXNG engines.

Prefix a query with `!ya` to use only the YaCy network.

## More information

- [DHT search configuration](../../../../services/yacydhtsearch/doc/configuration.md)
- [DHT search behavior](../../../../services/yacydhtsearch/doc/specification.md)
