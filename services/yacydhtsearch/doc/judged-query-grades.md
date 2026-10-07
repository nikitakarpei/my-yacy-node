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

A page that names its subject in one or two lines, and gives no more, gets the
grade `1`. Few documents of a pool reach `2`.

Never change a grade or a spam mark that a person gave.

## A document without stored page text

When the capture cannot fetch a page for a passing reason, such as a timeout,
a connection error, or the answer `429` or `5xx`, the document keeps the grade
`null` until a capture stores its page.

When the capture gets the answer `404` or `410`, the page is gone, and the
document gets the grade `0`.

When the site refuses the capture, such as with the answer `401`, `403` or
another `4xx`, or with a bot challenge, an error page or a consent wall in place
of the page, grade the document from a copy that a browser or a web archive
took. Use the copy that is nearest to the recording. Give at most the grade `1`
when that copy is more than one year older or newer than the recording. When
there is no such copy, grade the document from its title, its address and the
snippet of the peers, and give at most the grade `1`.

## The language of the query

When the query has a language, only a page in that language gets the grade
`2`. A page in a different language gets at most the grade `1`. This is also
true when the query is the name of a person, a site or a product.

When the query has no language, the language of the page does not change the
grade.

## The spam mark

`"spam": true` marks a page that deceives the reader or the search engine
about what it is: a doorway page, a link farm, a blog network page, a copy of
a site of a different owner, a reused domain, a seller of fake goods sold as
genuine, a template affiliate page, a review that claims to be independent but
is not, a page with hidden words its owner put there, a page stuffed with
keywords or links, or a page that a different person added to a real site.

The claims of a page, true or false, get no mark. Words, footer links or spam
comments that a different person put on a real page get no mark.

## Health claims

A page with health claims against medical knowledge gets the grade `0`.
