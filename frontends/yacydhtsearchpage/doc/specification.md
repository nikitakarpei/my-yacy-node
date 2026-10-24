# yacydhtsearchpage — Technical Specification

## Context

`yacydhtsearchpage` is a search page for the YaCy network. It shows HTML pages that the
server renders, in the layout of the search page of YaCy. It asks `yacydhtsearch` through
the public `/yacysearch.json` interface of YaCy, and it keeps no state of its own.

## Non-Goals

* Merging or ranking the results of more than one search service.
* Authenticating or authorizing a user.
* Showing texts in a language other than English.

## Functional Requirements

* The page SHALL show a search box.
* The page SHALL show, for a query, the results that `yacydhtsearch` finds, ten to a page.
* The page SHALL show the title, the description, and the address of each result, and its
  publication date when the result has one.
* The page SHALL link the next page when the shown page is full, and the previous page when
  the shown page is not the first.
* The page SHALL NOT link a result to a script address.
* When the operator names `visitcrawl`, the page SHALL link each web result through `/visit`
  on its own address with a link that `visitcrawl` accepts, and SHALL pass each visit on to
  `visitcrawl`.
* The page SHALL give a result the same visit link for the whole day.
* The page SHALL tell the user when `yacydhtsearch` does not answer.

## Non-Functional Requirements

* Every text that the page shows SHALL come from one message catalog.
* The page SHALL work without scripts in the browser.

## Known Limitations

* A visit link changes each day, so a result that you opened on an earlier day does not
  show as visited.
* The page does not know the number of results. It links a next page after each full page,
  also when no more results follow.
