# searchpage — Technical Specification

## Context

`searchpage` is a search page for a person who tries the peer. It searches the YaCy network
through `yacydhtsearch`, the crawled pages through `corpustext`, or both, and it shows the
results on one page that the server renders. A person who wants more sources or settings uses
SearXNG.

## Non-Goals

* Searching sources other than the YaCy network and the crawled pages.
* Settings that a person keeps, or a login.
* Images, news, maps, filters by date, or suggestions while a person types.
* Running JavaScript in the browser.
* Showing texts in a language other than English.

## Functional Requirements

* The service SHALL show the search box at `/` and at `/search`.
* The list of search operators SHALL be open before a search and closed on the results.
* When the operator sets a tab bar, each page SHALL show it at the top, with the tab of the
  search page marked.
* The service SHALL read the query from the `q` parameter and the page number from the `page`
  parameter. A page number that is not a positive number SHALL mean the first page.
* The service SHALL search only the sources that the operator sets, and SHALL NOT start
  without one.
* The operator `!ct` SHALL limit a search to the crawled pages, and `!ya` to the YaCy network.
  A query without these operators SHALL search every source that the operator sets.
* The operator `:xx` SHALL limit a search to pages in the language `xx`. A query without it
  SHALL search all languages.
* A word that starts with `!` and names no source that the operator sets SHALL stay a query word.
* The service SHALL show the results of the chosen sources in one list. Without a shared
  address, the list SHALL alternate the sources by position, the crawled pages first.
* The list SHALL show an address that both sources give once, no lower than the higher of its
  two places in the alternating list.
* The service SHALL show a notice for each chosen source that failed, above the results of the
  other source.
* The service SHALL say that there are no results only when every chosen source answered and
  none gave a result.
* The service SHALL show the query words in a snippet in bold.
* The service SHALL link the next page while a chosen source gave a full page of results, and
  the previous page after the first page.
* When the operator sets `visitcrawl`, a result that a person opens SHALL go through
  `visitcrawl` to the result page, and a person SHALL need to reach only the search page.
* When the operator does not set `visitcrawl`, a result SHALL link straight to its page.
* The service SHALL serve an OpenSearch description, so that a browser can add the page as a
  search engine.

## Non-Functional Requirements

* A search SHALL take no longer than the slowest chosen source, up to a bound for each source.
* The service SHALL NOT send the query to the page that a person opens from a result.

## Known Limitations

* There is no login. Anyone who reaches the page can search. With `visitcrawl`, they can also
  make the crawler visit the pages of the results.
* A search that asks the YaCy network waits for it, up to the time that the operator sets.
* The YaCy network gives at most the results that one ranking of `yacydhtsearch` holds.
* The page shows the titles that the YaCy network gives as they are.
* When the operator sets `visitcrawl` and it does not answer, a result link does not open.
