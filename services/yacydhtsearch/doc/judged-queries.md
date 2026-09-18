# Judged queries

The judged query set measures how well the ordering of the service puts the
documents that answer a query first. It holds 72 queries: one word, two words,
three or more words, navigational, other languages, and queries no peer answers.
Each query has three files in `test/judgedqueries/testdata/`, named by the query
words in lower case and joined by `-`: `answers/`, `judgments/`, `pagetext/`.

`judged-query-steps.md` tells how to record, derive, accept and tune the set.

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

The gate reports two more gains of each ordering, and asserts neither: the gain
that discounts a repeated subtopic in place of a repeated host, and the gain
that discounts no repetition. The three show what the host discount costs and
what it gives.

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

## How to judge the subtopic

A subtopic names which intent of the query a document answers, as `jaguar`
holds the intents `animal` and `car`. Give a subtopic to every document of the
grade 1 or more of a query that holds more than one intent. Leave the subtopic
out elsewhere: the host of a document then stands in for its subtopic.

Give the subtopic `other` to a document whose intent no other document of the
same query holds.
