# corpustext in Elasticsearch

## Index names

An index is named `<base>_v<version>_<language>`. The base is `ELASTICSEARCH_INDEX`. The
language is the primary subtag of a configured language, or `und` for all other pages.

A search client reads the corpus through the wildcard `<base>_v<version>_*`. The wildcard
also matches any other index with that prefix.

## Mapping

| Field | Type | Analyzer |
|---|---|---|
| `title` | `text` | `corpus_text` |
| `content` | `text` | `corpus_text` |
| `url` | `keyword` | — |
| `language` | `keyword` | — |
| `crawled_at` | `date` | — |

## Text analysis

The `corpus_text` analyzer of an index is the built-in analyzer of its language: `english`,
`german`, `french` or `russian`. In the `und` index it is the `standard` tokenizer with the
`lowercase` and `asciifolding` filters.

## Documents

The document identifier is the SHA-256 of the canonical URL, in hexadecimal. The service
writes it with `PUT /<index>/_doc/<identifier>`, which replaces the document of that URL.
