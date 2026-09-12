# 6. Drop stopwords with the Snowball lists

Date: 2026-09-12

## Status

Accepted

## Context

A client writes a query in plain words. Words such as `how`, `do` and `my` occur in almost
every page, so they ask many peers for postings that tell the results apart in no way, and
they lift the score of a page that holds them often. The query words also count in the page
text and choose the snippet.

## Decision

The service embeds the stopword lists of the Snowball project for English, German, French,
Spanish, Italian and Russian as data files, with the comments removed and the BSD-3 license
of Snowball beside them. No module carries the lists alone; the modules that hold them also
hold a stemmer, a tokenizer or a language model that the service does not use. The query
keeps the words outside the list of its language. The list is the one the `lr` field of the
client names, or, when the client names none, the one that covers the most query words.
Two lists that cover as many words, or a list that covers none, leave the query as it is,
and a query of stopwords alone keeps every word.

## Consequences

A query in one of the six languages asks the peers for the words that carry its subject.
A query in another language keeps its stopwords, and so does a query of two words that two
lists cover as often. A word that is a stopword in one language and carries meaning in
another, such as `die`, goes away when the words around it point to the first language.
