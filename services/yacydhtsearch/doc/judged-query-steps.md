# Judged query steps

The judged query set in `test/judgedqueries/testdata/` measures how well the
ordering of the service puts the documents that answer a query first.
`judged-queries.md` tells what the set holds and how to grade it. Each step
here runs from `services/yacydhtsearch`.

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
