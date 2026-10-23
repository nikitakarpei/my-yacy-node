# yacycrawler — API Specification

## Context

An operator runs crawls through the operator API of `yacycrawler`, over HTTP. The operator
places an order, follows each crawl that orders start, and stops or holds crawling.
Machines continue to place orders on the orders stream. The service serves the API only
when an API address is configured, and every instance answers every request.

## Conventions

* Bodies are JSON, of the types in the crawler contract, `yacycrawlcontract`.
* 400 carries the reason as text. 404 means the service knows no crawl of this ID.
* A list takes `limit` and `cursor`. It answers at most `limit` items, and a cursor for
  each neighbor page that exists. A `limit` outside its range answers 400.
* The count of pending page visits is exact. The progress counts can count a visit twice
  or not at all.
* A request that changes state from another origin answers 403. There is no login.

## Endpoints

### `PUT /orders/{id}` — place an order

Takes a `CrawlOrder`. The first order of an ID starts one crawl; a later one changes nothing
while its crawl is kept.

* 201, 200: the `Crawl` the order started or found, under the crawl ID the service gives it.
* 400: the order is not readable, has no seed URLs, or has an ID other than `{id}`.

### `GET /crawls` — list crawls

Answers a `CrawlPage`, from the newest start to the oldest. Each `state` parameter adds a
crawl state to the filter. The cursor comes from `OlderCursor` or `NewerCursor`.

### `GET /crawls/{id}` — show a crawl

Answers the `Crawl`: its state, start and finish, pending page visits, completed visits by
result, dropped visits by reason, and the links its profile did not admit.

### `GET /crawls/{id}/hosts` — list the hosts of a crawl

Answers a `CrawlHostPage`, the host with the most URLs first, each with its pending page
visits, completed visits by result, and dropped visits by reason. The cursor comes from
`NextCursor` or `PreviousCursor`.

### `POST /crawls/{id}/cancel` — cancel a crawl

* 204: the crawl takes no more page visits. It shows as cancelling while it drops its
  pending page visits, and then as cancelled.
* 409: the crawl succeeded, and stays succeeded.

### `PUT /crawls/{id}/pause` — pause or resume a crawl

Takes a `Pause`: `Paused` true pauses the crawl, false resumes it. A paused crawl keeps its
pending page visits and takes none until it is resumed.

* 200: the `Pause`.
* 409: the crawl is cancelling or finished.

### `GET /crawler-status` — show the crawler

Answers the `CrawlerStatus`: the pause of every page visit, the pending page visits, and
the amount of running crawls.

### `PUT /crawler-pause` — pause or resume every page visit

Takes a `Pause` and answers it with 200. The state of each crawl does not change, and order
intake continues to put page visits on the frontier.

## Known Limitations

* A crawl with a large backlog delays the page visits of every other crawl.
* A crawl keeps one record for each URL it admits until it is forgotten, so a wide crawl
  grows the crawl state. Forgetting it is one large write that other writes wait for.
* A forgotten crawl leaves zero counts of its pending page visits in the crawl state.
* The crawl list shows the one-page crawls of `visitcrawl` with the crawls of operators.
* A crawl is kept only for the crawl retention after it finishes.
