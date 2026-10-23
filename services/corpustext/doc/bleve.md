# corpustext in Bleve

## Index names

An index is the directory `<BLEVE_DIRECTORY>/yacy_text_v<version>_<language>`. The
language is the primary subtag of a configured language, or `und` for all other pages.
The service searches all indexes of its version at once.

## Text analysis

An index uses the Bleve analyzer of its language: `en`, `de`, `fr` or `ru`. The `und`
index splits words, folds accents, and lowercases.

The service creates an index only when it is absent. It does not change an index that
exists.

## Documents

Writing a page replaces the document of its canonical URL.
