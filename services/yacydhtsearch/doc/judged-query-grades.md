# Judged query grades

Each document of a judged query has a grade. The grade tells how well the page
answers the query. A document is judged when it has a grade, when its page text
is stored, or when the found order or the ordering of the service puts it in
its first ten. A new document gets `null`.

Grade a document from the stored page text, the title and the address.

## The grades

- `2` — the page answers the query. For a query that names a site or a
  product, only the page the name points at gets `2`, not its other pages.
- `1` — the subject of the query is a main topic of the page, but the page
  does not answer it.
- `0` — the page has nothing to do with the query, only shares a word with it,
  lists many subjects as a tag page does, or is spam, even on the subject.

A document with no stored page text gets at most the grade `1`. A page that
names its subject in one or two lines, and gives no more, gets the grade `1`.
Few documents of a pool reach `2`.

Never change a grade or a spam mark that a person gave.

## The language of the query

When the query is prose, only a page in the language of the query gets the
grade `2`. A page in a different language gets at most the grade `1`.

When the query is the name of a person, a site or a product, the language of
the page does not change the grade.

## The spam mark

The spam mark protects the person who searches. A page gets the mark for one of
two causes: the page deceives the reader about what it is, or the page can
cause harm to the health of the reader.

Give `"spam": true` to a page that deceives the reader: a fake page, a doorway
page, a hijacked domain, a link farm, or a seller of medicines. Give the mark
also to a page that speaks against vaccines in its own text, because such a
page can cause the reader to refuse a medical treatment.

Give no spam mark for a different opinion, such as a text against the climate
science or against a historical record. Such a page can be incorrect, but it
shows the reader what it is, and it gives no medical advice.

Give no spam mark when the page shows such headlines only in a list, in a
sidebar, or in an archive of a month. Give no spam mark to a page that reports
on the persons who make the claims.
