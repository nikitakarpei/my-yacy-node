# Judged queries

The judged query set measures how well the ordering of the service puts the
documents that answer a query first. It holds 42 queries in six groups: one
word, two words, three or more words, navigational, other languages, and
queries that no peer can answer.

For each query the set holds these files in `test/judgedqueries/testdata/`. The
words of the query, in lower case and joined by `-`, make each `<query>` name.

- `answers/<query>.json` holds what the peers answered for the query.
- `judgments/<query>.json` holds the grade of each judged document.
- `pagetext/<query>/<document>.txt.gz` holds the text of one page.

## How to record the answers again

The recorder asks the peers, reads the page of each of the first fifty
documents of the relevance ordering, and stores the text of each page it read.
From that text it writes the hits, the query phrase hits, the amount of words
and the snippet of each document. A document whose page the recorder did not
read keeps the counts of the peers and gets no snippet.

The recorder asks the live freeworld network from this host and reads the pages
from the web, and needs direct egress for both. The run takes some
minutes, writes every file again, and gives how many grades each query waits
for. Run it from `services/yacydhtsearch`:

```sh
YACYDHTSEARCH_RECORD_JUDGED_QUERIES=1 go test \
    -run TestRecordWhatThePeersAnswerForTheJudgedQueries \
    -timeout 40m -v ./test/judgedqueries/
```

## How to derive the answers again

A change of the rule that reads the text of a page needs no new recording. This
step writes the hits, the query phrase hits, the amount of words and the
snippet of each answers file again from the stored page text. It asks no peer
and no page. Run it from `services/yacydhtsearch`:

```sh
YACYDHTSEARCH_DERIVE_JUDGED_QUERIES=1 go test \
    -run TestDeriveTheJudgedQueriesFromTheStoredPageText \
    -v ./test/judgedqueries/
```

## How to grade

The recorder and the derivation write the judgments files. A document is judged
when its page text is stored, or when the peer ordering puts it in its first
ten. Both steps keep each grade that a person gave, and give each new document
the grade `null`. Give each document with the grade `null` one grade:

- `2` — the page answers the query.
- `1` — the page is about the subject of the query, but does not answer it.
- `0` — the page has nothing to do with the query.

Grade from the stored text of the page, the title and the address. A document
with no stored text gets the grade that the title and the address support,
which is `1` at most. Never change a grade that a person gave.

## What the gate asserts

`TestTheRelevanceOrderingHoldsItsGainOverTheJudgedQueries` measures the
normalized discounted cumulative gain of the first ten graded documents of the
ordering of the service and of the peer ordering, and takes the mean of each
over the queries. The ordering of the service is the relevance ordering where
the hosts take turns.

The gate drops a document with the grade `null` before it measures, so a change
of the ordering needs no new grading. A query whose judgments hold no document
of grade 1 or more does not count. The mean of the ordering of the service must
stay at or above the floor that the test holds, and at least the lift that the
test holds above the mean of the peer ordering.

## Limits

The recorded answers and the stored page text are a photograph of the network
and of the web on the day of the recording. The gain of a live search is not
the gain this gate measures.
