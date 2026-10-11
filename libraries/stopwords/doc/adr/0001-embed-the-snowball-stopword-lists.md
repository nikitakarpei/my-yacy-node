# 1. Embed the Snowball stopword lists

Date: 2026-10-11

## Status

Accepted

## Context

Words such as `the`, `und` and `de` occur in almost every page. The search
service leaves them out of a query, as its ADR 0006 records. The node can leave
them out of the postings it stores. Both services need the same list for each
language, and a service that keeps its own copy can drift from the other.

## Decision

The library embeds the stopword lists of the Snowball project for English,
German, French, Spanish, Italian and Russian as data files, with the comments
removed and the BSD-3 license of Snowball beside them. No module carries the
lists alone; the modules that hold them also hold a stemmer, a tokenizer or a
language model that the services do not use.

The library tells whether a word is a stopword of a language, gives all the
stopwords of a language, and names the one language whose list covers the most
of a set of words. What a service does with a stopword stays in that service.

## Consequences

A service that uses the library needs no ADR of its own for the lists. A
language without a list has no stopwords. A change to a list changes every
service that uses the library at its next build.
