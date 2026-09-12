# Judged queries

The judged query set measures how well the ordering of the service puts the
documents that answer a query first. It holds 72 queries in six groups: one
word, two words, three or more words, navigational, other languages, and
queries that no peer can answer.

Each query has three files in `test/judgedqueries/testdata/`, named by the
words of the query in lower case and joined by `-`: the answers of the peers in
`answers/`, the grade of each judged document in `judgments/`, and the text of
each page in `pagetext/`.

## The gate

`TestTheRelevanceOrderingHoldsItsGainOverTheJudgedQueries` measures the first
ten graded documents of each ordering. The gain of a document is its grade,
discounted by its place, and discounted by half again for each document of its
host above it that has the grade 1 or more. The gate divides by the gain of the
ideal order and drops a document of the grade `null`.

The mean of the ordering of the service must stay at or above the floor of the
test, and at least the lift of the test above the mean of the peer ordering. A
live search reaches another gain, because the fixtures are a photograph of the
day of the recording.

## How to grade

A document is judged when its page text is stored, or when the peer ordering
puts it in its first ten. Each new document gets the grade `null`. Grade it
from the stored page text, the title and the address:

- `2` — the page answers the query.
- `1` — the page is about the subject of the query, but does not answer it.
- `0` — the page has nothing to do with the query.

A document with no stored text gets at most the grade `1`. Never change a grade
a person gave.

## How to record the answers again

The recorder asks the live freeworld network from this host and reads the pages
of the first fifty documents from the web, so it needs direct egress for both.
It writes every file again and takes some minutes.

```sh
YACYDHTSEARCH_RECORD_JUDGED_QUERIES=1 go test \
    -run TestRecordWhatThePeersAnswerForTheJudgedQueries \
    -timeout 40m -v ./test/judgedqueries/
```

## How to derive the answers again

A change of the rule that reads the text of a page needs no new recording. This
step writes the hits, the query phrase hits, the amount of words and the
snippet of each answers file again from the stored page text.

```sh
YACYDHTSEARCH_DERIVE_JUDGED_QUERIES=1 go test \
    -run TestDeriveTheJudgedQueriesFromTheStoredPageText \
    -v ./test/judgedqueries/
```

## How to tune the score weights

This step searches the weight of each score of the relevance ordering over a
grid of values and gives the weights of the highest mean gain. It also tunes on
one half of the queries and measures on the other half, which holds the same
six groups. It changes no file and takes some minutes.

```sh
YACYDHTSEARCH_TUNE_RELEVANCE_WEIGHTS=1 go test \
    -run TestTuneTheScoreWeightsOfTheRelevanceOrdering \
    -timeout 30m -v ./test/judgedqueries/
```

Run each step from `services/yacydhtsearch`.
