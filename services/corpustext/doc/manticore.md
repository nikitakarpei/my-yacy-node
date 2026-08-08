# corpustext in Manticore

## Table names

A table is named `<base>_v<version>_<language>`. The base is `MANTICORE_TABLE`. The
language is the primary subtag of a configured language, or `und` for all other pages.

A search client reads the corpus through the distributed table `<base>_v<version>`. The
service recreates that table at every start, with the configured tables as its members.

## Columns

| Column | Type |
|---|---|
| `title` | `text` |
| `content` | `text` |
| `url` | `string` |
| `language` | `string` |
| `crawled_at` | `string` |

## Text analysis

The morphology of a table is the stemmer of its language: `stem_en`, `libstemmer_de`,
`libstemmer_fr` or `libstemmer_ru`. The `und` table has no morphology.

## Documents

The document identifier is the first 64 bits of the SHA-256 of the canonical URL, shifted
right by one bit to stay positive. The service writes a document with `POST /replace`,
which replaces the document of that URL.
