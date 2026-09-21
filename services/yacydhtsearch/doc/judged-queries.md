# Judged queries

The judged query set measures how well the ordering of the service puts the
documents that answer a query first. It holds 80 queries: one word, two words,
three or more words, navigational, other languages, and queries no peer answers.
Each query has three files in `test/judgedqueries/testdata/`, named by the query
words in lower case and joined by `-`: `answers/`, `judgments/`, `pagetext/`.

## The gate

`TestTheRelevanceOrderingHoldsItsGainOverTheJudgedQueries` measures the first
ten graded documents of each ordering. The gain of a document is its grade,
discounted by its place, and discounted by half again for each document of its
site above it that has the grade 1 or more. A site is the host of the address
without the world wide web label. The gate divides by the gain of the ideal
order and drops a document of the grade `null`.

The mean gain must stay at least the lift of the test above the found order,
the order in which the service found the documents. It must also stay at or
above the mean in `testdata/accepted-gain-per-judged-query.json`, less the
tolerance of the test, over the queries that both hold. A query accepted above
zero must not fall to zero. The gate logs the spam documents in each first ten.

## How to grade

`judged-query-grades.md` gives the grades, the rule for the language of the
query, and the spam mark.

## How to record the answers again

The recorder asks the live freeworld network and reads the pages of the first
fifty documents from the web. It needs egress and writes every file again. A
`-run` pattern that ends in the file name of one query records only that query.

```sh
YACYDHTSEARCH_RECORD_JUDGED_QUERIES=1 go test -timeout 40m -v \
    -run TestRecordWhatThePeersAnswerForTheJudgedQueries ./test/judgedqueries/
```

## How to derive the answers again

This step writes the contents of every read page of each answers file again
from the stored page: the title, the snippet, the hits, the query phrase hits,
the amount of words and the links of each kind.

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

This step gives the score weights of the highest mean gain over a grid. It tunes
on one half of the queries to measure on the other, and changes no file.

```sh
YACYDHTSEARCH_TUNE_RELEVANCE_WEIGHTS=1 go test -timeout 30m -v \
    -run TestTuneTheScoreWeightsOfTheRelevanceOrdering ./test/judgedqueries/
```

Run each step from `services/yacydhtsearch`.
