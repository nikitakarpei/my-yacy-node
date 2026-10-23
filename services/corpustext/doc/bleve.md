# corpustext in Bleve

## Index names

An index is the directory `<BLEVE_DIRECTORY>/yacy_text_v<version>_<language>`. The
language is the primary subtag of a configured language, or `und` for all other pages.

The service searches all indexes of its version at once.

## Fields

| Field | Type | Stored |
|---|---|---|
| `title` | text | yes |
| `content` | text | yes |
| `auxiliary_text` | text | no |
| `url` | keyword | yes |
| `language` | keyword | no |
| `crawled_at` | date | no |

The `content` field holds the readable text of the page. The `auxiliary_text`
field holds the text that the page has beside its article, such as notes.

## Text analysis

The analyzer of an index is the Bleve analyzer of its language: `en`, `de`, `fr` or `ru`.
In the `und` index it is the `unicode` tokenizer with ASCII folding and lowercase.

The service creates an index only when it is absent. It does not change an index that
exists.

## Documents

The document identifier is the canonical URL. Writing a document replaces the document
of that URL.
