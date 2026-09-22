# Judged queries

The judged query set measures how well the ordering of the service puts the
documents that answer a query first. It holds 120 queries: of one word, of two,
of three or more, navigational, in other languages, and queries no peer answers.
Each query has three files in `test/judgedqueries/testdata/`, named by the query
words in lower case and joined by `-`: `answers/`, `judgments/`, `pages/`.
`judged-query-grades.md` gives the grades, the language rule and the spam mark.

## The gate

`TestTheRelevanceOrderingHoldsItsGainOverTheJudgedQueries` measures the first
ten graded documents of each ordering. The gain of a document is its grade,
discounted by its place, and discounted by half again for each document of its
site above it that has the grade 1 or more. A site is the host of the address
without the world wide web label. The gate divides by the gain of the ideal
order and drops a document of the grade `null`.

The mean gain must stay at least the lift of the test above the found order. It
must also stay at or above the mean in
`testdata/accepted-gain-per-judged-query.json`, less the tolerance of the test,
over the queries that both hold. A query accepted above zero must not fall to
zero. The gate logs the spam documents in each first ten.

## How to write the fixtures again

Run each step from `services/yacydhtsearch`. The steps come in this order: a
recording gives the answers, a capture gives the pages, and the derivation
gives the contents of the pages to the answers and the judgments.

The recorder asks the live freeworld network and reads the pages of the first
fifty documents from the web. It needs egress and writes every file again. A
`-run` pattern that ends in the file name of one query records only that query.

```sh
YACYDHTSEARCH_RECORD_JUDGED_QUERIES=1 go test -timeout 40m -v \
    -run TestRecordWhatThePeersAnswerForTheJudgedQueries ./test/judgedqueries/
```

The capture reads from the web the page of each document a judgments file
names, and writes the pages file of every query again. It needs egress.

```sh
YACYDHTSEARCH_CAPTURE_JUDGED_QUERY_PAGES=1 go test -timeout 40m -v \
    -run TestCaptureThePagesOfTheJudgedQueries ./test/judgedqueries/
```

The derivation writes from the stored page the title, the snippet, the hits,
the query phrase hits, the amount of words and the links of each kind. It keeps
the time of the recording.

```sh
YACYDHTSEARCH_DERIVE_JUDGED_QUERIES=1 go test -v \
    -run TestDeriveTheJudgedQueriesFromTheStoredPages ./test/judgedqueries/
```

## How to accept a new baseline

This step writes the gain of each judged query to the baseline file again.

```sh
YACYDHTSEARCH_ACCEPT_JUDGED_QUERIES_BASELINE=1 go test -v \
    -run TestAcceptTheGainOfEachJudgedQueryAsTheBaseline ./test/judgedqueries/
```

## How to tune the relevance weights

This step gives the relevance weights of the highest mean gain over a grid. It
tunes on one half of the queries to measure on the other, and changes no file.

```sh
YACYDHTSEARCH_TUNE_RELEVANCE_WEIGHTS=1 go test -timeout 30m -v \
    -run TestTuneTheRelevanceWeights ./test/judgedqueries/
```
