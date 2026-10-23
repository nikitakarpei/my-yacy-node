# corpustext in Manticore

## Table names

A table is named `<base>_v<version>_<language>`. The base is `MANTICORE_TABLE`. The
language is the primary subtag of a configured language, or `und` for all other pages.

The service searches the corpus through the distributed table `<base>_v<version>`. The
service recreates that table at every start, with the configured tables as its members.

## Columns

| Column | Type |
|---|---|
| `title` | `text` |
| `content` | `text` |
| `auxiliary_text` | `text` |
| `url` | `string` |
| `language` | `string` |
| `crawled_at` | `string` |

The `content` column holds the readable text of the page. The `auxiliary_text`
column holds the text that the page has beside its article, such as notes. At start,
the service adds each missing column to a table that exists.

## Text analysis

The morphology of a table is the stemmer of its language: `stem_en`, `libstemmer_de`,
`libstemmer_fr` or `libstemmer_ru`. The stopwords of a table are the built-in Manticore
list of its language. Search and highlight ignore stopwords. The `und` table has no
morphology and no stopwords.

The service creates a table only when it is absent. It does not change the text
analysis of a table that exists.

## Documents

The document identifier is the first 64 bits of the SHA-256 of the canonical URL, shifted
right by one bit to stay positive. The service writes a document with `POST /replace`,
which replaces the document of that URL.
