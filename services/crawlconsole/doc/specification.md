# crawlconsole — Technical Specification

## Context

`crawlconsole` is a web console for the operator of `yacycrawler`. It shows HTML pages that
the server renders. It reads and changes crawls only through the operator API of
`yacycrawler`, and it keeps no state of its own.

## Non-Goals

* Authenticating or authorizing an operator.
* Placing orders on the orders stream.
* Showing a crawl that the crawler no longer keeps.
* Showing texts in a language other than English.

## Functional Requirements

* The console SHALL show the running and paused crawls in one table and the finished
  crawls in a second table, each from the newest start, in pages.
* The console SHALL turn the pages of each table separately.
* The console SHALL show one crawl with its state, its duration, its links not followed,
  and its order.
* The console SHALL show the pending visits of a running crawl and the other visits grouped
  by what became of them, in total and for each host, the host with the most URLs first.
* The console SHALL let an operator pause, resume, and cancel a crawl that did not finish.
* The console SHALL start a crawl from a form, under an order ID that it makes for each
  form that it sends. It SHALL show the reason when the crawler refuses the order.
* The console SHALL fill the form from the order of a crawl that the operator duplicates.
* The console SHALL show in each page header whether the crawler crawls, is idle, or is
  paused, and SHALL let the operator pause and resume every page visit.
* The console SHALL refuse a form that comes from another origin.
* The console SHALL tell the operator when the crawler does not answer.

## Non-Functional Requirements

* Every text that the console shows SHALL come from one message catalog.
* The crawl tables, and a crawl that did not finish, SHALL refresh by themselves.

## Known Limitations

* There is no login. Anyone who reaches the console can place, cancel, and pause crawls.
* A pause, resume, or cancel that fails shows no notice. The page shows the crawl state
  that the crawler keeps.
* A crawler word that the catalog does not know stops the page that shows it.
