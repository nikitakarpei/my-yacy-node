# yacycrawler — Operation Specification

## Context

An operator runs crawls through the operator API of `yacycrawler`, over HTTP. The operator
places an order, follows each crawl that orders start, and stops or holds crawling.
Machines continue to place orders on the orders stream. The request and answer types are
part of the crawler contract, `yacycrawlcontract`.

## Non-Goals

* Authenticating or authorizing an operator.
* Keeping a crawl longer than the crawl retention after it finishes.
* Sharing page visits fairly between crawls.

## Functional Requirements

* The service SHALL serve the operator API only when an API address is configured.
* An operator SHALL place an order under an ID the operator chooses. The service SHALL
  refuse an order it cannot read, SHALL start one crawl for each ID, and SHALL answer with
  that crawl under the crawl ID the service gives it.
* The service SHALL list crawls from the newest start to the oldest, in pages before or
  after a crawl, and SHALL filter the list by crawl state.
* Each crawl SHALL show its state, its start and finish, its pending page visits, its
  completed visits by result, its dropped visits by reason, and the links its profile
  did not admit.
* The service SHALL list the hosts of a crawl, the host with the most URLs first, each with
  its pending page visits, its completed visits by result, and its dropped visits by reason.
* An operator SHALL cancel a running or paused crawl. The crawl SHALL take no more page
  visits, SHALL show as cancelling while it drops its pending page visits, and then SHALL
  show as cancelled. A succeeded crawl SHALL stay succeeded.
* An operator SHALL pause and resume one crawl. A paused crawl SHALL keep its pending page
  visits and take none until it is resumed. A cancelling or finished crawl SHALL refuse
  both.
* An operator SHALL pause and resume every page visit. This pause SHALL NOT change the
  state of a crawl.
* The service SHALL refuse a request that changes state when it comes from another origin.

## Non-Functional Requirements

* Every instance SHALL answer every request; an instance keeps no crawl state of its own.
* The count of pending page visits SHALL be exact. The progress counts can count a visit
  twice or not at all.

## Known Limitations

* There is no login. Anyone who reaches the API address can place, cancel, and pause
  crawls.
* A crawl with a large backlog delays the page visits of every other crawl.
* A crawl keeps one record for each URL it admits until it is forgotten, so a wide crawl
  grows the crawl state.
* Forgetting a wide crawl is one large write, and other writes of the crawl state wait
  for it.
* A cancelling crawl drops its pending page visits only while the crawler takes page
  visits, so a full crawled page backlog delays it.
* A forgotten crawl leaves zero counts of its pending page visits in the crawl state.
* The crawl list shows the one-page crawls of `visitcrawl` with the crawls of operators.
* A pause of every page visit does not stop order intake. Orders continue to put page
  visits on the frontier.
