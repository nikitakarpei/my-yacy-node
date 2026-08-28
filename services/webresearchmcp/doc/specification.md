# webresearchmcp — Technical Specification

## Context

An assistant that speaks the Model Context Protocol has no way into the operator's own
stack: it cannot search the operator's SearXNG, and it cannot read a page as markdown.
`webresearchmcp` is a separate, optional, disposable Go service that serves two tools over
MCP. One tool searches the web. The other tool answers with the markdown of one page, and
asks the stack to scrape that page first.

## Non-Goals

* Fetching, parsing, or storing a page itself.
* Ranking, re-ranking, or filtering the results SearXNG returns.
* Crawling more than the one page a caller asks for, or following links on it.
* Turning a search result into a crawl order.
* Keeping a history of searches or of the pages callers ask for.
* Authenticating callers, or limiting how often they call.

## Functional Requirements

* The service SHALL serve two tools over MCP: one that searches the web, and one that
  answers with the markdown of one page.
* A search SHALL answer with the results the configured SearXNG returns, in the order
  SearXNG returns them.
* A search SHALL ask the configured SearXNG to leave the result links as they are.
* A search answer SHALL carry the destination link of each result.
* A page call SHALL ask for a scrape of that page, unless it names a version.
* A page call SHALL wait for the scrape it asked for, and SHALL stop waiting when that
  scrape ends or when the wait ends, whichever comes first.
* A page call SHALL answer with the markdown the corpus holds for the page when it stops
  waiting.
* A page call SHALL answer that no markdown is available if the corpus holds none.
* A page answer SHALL carry the URL of the page, the version of the page, and the time the
  corpus stored that version.
* A page answer SHALL say whether the page was read from the web for this call.
* A page answer SHALL carry the first characters of the markdown, up to the limit for that
  call.
* A page answer that carries less than the whole markdown SHALL say so, and SHALL carry how
  many characters the whole markdown has.
* A page call MAY name its own limit, which SHALL replace the configured one.
* A page call MAY name the version it read before, to read more of that same version.
* A page answer to a call that names a version SHALL say whether it carries that version.
* The service SHALL let operators configure the SearXNG it searches, the broker it asks for
  scrapes, and the corpus it reads markdown from.
* The service SHALL let operators configure how long it waits for a scrape and the limit a
  page answer carries.

## Non-Functional Requirements

* The service SHALL bound every search and every page call by a deadline, whatever the state
  of SearXNG, of the broker, and of the corpus.
* The service SHALL persist no state of its own between calls.
* The service SHALL keep memory usage bounded independently of call volume, with an explicit
  limit on calls in flight.
* SearXNG and the markdown corpus SHALL each sit behind a narrow interface, so either can be
  replaced without a change to the tools.
* The service SHALL expose its behavior as machine-readable metrics, including searches
  served, pages answered, and pages answered without a read from the web.

## Known Limitations

* The wait can end before the corpus stores the page, because the scrape is slow or because
  the service does not learn that it ended. The caller then gets the version the corpus held
  before, although the page can be stored soon after.
* A page answer carries only the start of a page. A caller that wants more must ask again
  with a larger limit.
* Each call that names no version asks for a scrape again, so repeated calls make the
  origin serve that page more than once.
* The corpus holds only the newest markdown of a page. A caller that asks for a version the
  corpus no longer holds cannot read that version again.
