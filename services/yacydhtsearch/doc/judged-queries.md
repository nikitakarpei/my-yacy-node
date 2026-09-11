# Judged queries

The judged query set measures how well the relevance ordering puts the
documents that answer a query first. It holds 42 queries in six groups: one
word, two words, three or more words, navigational, other languages, and two
queries that no peer can answer.

For each query the set holds two files in
`test/judgedqueries/testdata/`:

- `answers/<query>.json` holds what the peers answered for the query. The
  words of the query, in lower case and joined by `-`, make the file name.
- `judgments/<query>.json` holds the graded pool of the query.

## How to record the answers again

The recorder follows the service: it asks the peers, puts the answers in the
relevance order, reads the page of each of the first fifty documents, and
writes what the page says as the hits, the amount of words and the snippet of
the document. A page that the recorder cannot read leaves the counts of its
document as the peers answered them, and leaves its snippet empty.

The recorder asks the live freeworld network from this host, and reads the
pages from the web. It needs direct egress for both. Run it from
`services/yacydhtsearch`:

```sh
YACYDHTSEARCH_RECORD_JUDGED_QUERIES=1 go test \
    -run TestRecordWhatThePeersAnswerForTheJudgedQueries \
    -timeout 30m -v ./test/judgedqueries/
```

The run takes about ten minutes. It writes every file in `answers/` again. In
`judgments/` it keeps each grade that a person gave, adds each newly pooled
document with the grade `null`, and removes each document that left the pool.

## How to grade

The pool of a query is the union of the first ten documents of the relevance
ordering and the first ten documents of the peer ordering. Give each pooled
document one of three grades:

- `2` — the page answers the query.
- `1` — the page is about the subject of the query, but does not answer it.
- `0` — the page has nothing to do with the query.

Grade from the text of the page. Fetch the page from the web at the time of
the grading and read it. A page that you cannot fetch gets the grade that the
title and the address support, which is `1` at most. A document that the file
does not name counts as `0`. A grade of `null` also counts as `0`, and shows
that the document still needs a grade.

## What the gate asserts

`TestTheRelevanceOrderingHoldsItsGainOverTheJudgedQueries` runs the relevance
ordering and the peer ordering over every recorded answer and measures the
normalized discounted cumulative gain of the first ten documents. It takes the
mean of each ordering over the queries. A query whose judgments hold no
document of grade 1 or more does not count towards the means.

The gate asserts two things. The mean of the relevance ordering must stay at or
above the floor that the test holds. The mean of the relevance ordering must
also stay at least the lift that the test holds above the mean of the peer
ordering. Each assertion fails with its own message, and each message reports
both means and the amount of counted queries.

## Limits

The recorded answers are a photograph of the network. The live network gives
other documents for the same query, so the gain of a live search is not the
gain this gate measures. Record the answers again, and grade the new documents,
when the measurement must stand for the network of today.

The page that the grader fetches is the page of today. The peer indexed the
page of an earlier day. A grade can therefore describe a text that the peer
never saw.

The pool holds only documents that one of the two orderings put in its first
ten. A document that answers the query, but that both orderings put lower,
stays ungraded and counts as 0.
