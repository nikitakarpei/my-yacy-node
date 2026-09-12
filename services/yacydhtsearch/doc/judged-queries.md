# Judged queries

The judged query set measures how well the ordering of the service puts the
documents that answer a query first. It holds 72 queries: one word, two words,
three or more words, navigational, other languages, and queries no peer answers.
Each query has three files in `test/judgedqueries/testdata/`, named by the query
words in lower case and joined by `-`: `answers/`, `judgments/`, `pagetext/`.

## The gate

`TestTheRelevanceOrderingHoldsItsGainOverTheJudgedQueries` measures the first
ten graded documents of each ordering. The gain of a document is its grade,
discounted by its place, and discounted by half again for each document of its
host above it that has the grade 1 or more. The gate divides by the gain of the
ideal order and drops a document of the grade `null`.

The mean gain must stay at least the lift of the test above the peer ordering.
The gate also compares the ordering of the service against the baseline in
`testdata/accepted-gain-per-judged-query.json`. The mean gain over the queries
that both hold must stay at or above the accepted mean less the tolerance of
the test, and a query accepted above zero must not fall to zero.

## How to grade

A document is judged when its page text is stored, or when the peer ordering
puts it in its first ten. Each new document gets the grade `null`. Grade it
from the stored page text, the title and the address:

- `2` — the page answers the query. For a query that names a site or a
  product, only the page the name points at gets `2`, not its other pages.
- `1` — the subject of the query is a main topic of the page, but the page
  does not answer it.
- `0` — the page has nothing to do with the query, or only shares a word with
  it, or only lists many subjects, as a tag page does.

A document with no stored text gets at most the grade `1`, and few documents
of a pool reach `2`. Never change a grade a person gave.

## How to record the answers again

The recorder asks the live freeworld network and reads the pages of the first
fifty documents from the web. It needs egress and writes every file again. A
`-run` pattern that ends in the file name of one query records only that query.

```sh
YACYDHTSEARCH_RECORD_JUDGED_QUERIES=1 go test -timeout 40m -v \
    -run TestRecordWhatThePeersAnswerForTheJudgedQueries ./test/judgedqueries/
```

## How to derive the answers again

This step writes the hits, the query phrase hits, the amount of words and the
snippet of each answers file again from the stored page text.

```sh
YACYDHTSEARCH_DERIVE_JUDGED_QUERIES=1 go test -v \
    -run TestDeriveTheJudgedQueriesFromTheStoredPageText ./test/judgedqueries/
```

## How to accept a new baseline

This step writes the gain of each judged query to the baseline file again.

```sh
YACYDHTSEARCH_ACCEPT_JUDGED_QUERIES_BASELINE=1 go test -v \
    -run TestAcceptTheGainOfEachJudgedQueryAsTheBaseline ./test/judgedqueries/
```

## How to tune the score weights

This step searches the weight of each score of the relevance ordering over a
grid of values, gives the weights of the highest mean gain, and tunes on one
half of the queries to measure on the other. It changes no file.

```sh
YACYDHTSEARCH_TUNE_RELEVANCE_WEIGHTS=1 go test -timeout 30m -v \
    -run TestTuneTheScoreWeightsOfTheRelevanceOrdering ./test/judgedqueries/
```

Run each step from `services/yacydhtsearch`.
