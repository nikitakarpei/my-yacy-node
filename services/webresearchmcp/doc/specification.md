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
* A search answer SHALL carry the link each result points at, and not a link that routes
  the caller through the operator's stack.
* A search answer SHALL carry at most the configured number of results, taken from the
  start of what SearXNG returns.
* A search MAY name its own number of results, which SHALL replace the configured one.
* A page call SHALL ask for a scrape only when the corpus holds no markdown for the page,
  or holds markdown older than the age the call tolerates.
* A page call that names a version SHALL ask for no scrape.
* A page call SHALL wait for the scrape it asked for, until it ends or the wait runs out.
* A page call SHALL answer with the markdown the corpus holds when it stops waiting.
* A page call SHALL answer that no markdown is available if the corpus holds none.
* A page call MAY name its own character limit or tolerated age, which SHALL replace the
  configured one.
* A tolerated age smaller than the configured one SHALL NOT replace it.
* A page call MAY name the version it read before, to read more of that same version.
* A page answer SHALL carry the URL of the page, the version the corpus holds for it, and
  the time the corpus stored that version.
* A page answer SHALL say whether the call fetched the page, could not read it, ran out of
  wait, or needed no fetch.
* A page answer SHALL carry the first characters of the markdown, up to the call's limit.
* An answer that carries less than the whole markdown SHALL say so, and SHALL carry how many
  characters the whole markdown has.
* The service SHALL let operators configure every dependency it reaches, every deadline it
  applies, and every limit an answer carries.

## Non-Functional Requirements

* The service SHALL bound every search and every page call by a deadline, whatever the state
  of SearXNG, of the broker, and of the corpus.
* The service SHALL persist no state of its own between calls.
* The service SHALL keep memory usage bounded independently of call volume.
* The service SHALL expose its behavior as machine-readable metrics, including searches
  served and pages answered by what became of their fetch.

## Known Limitations

* The wait can end before the corpus stores the page. The caller then gets the version the
  corpus held before, although the page can be stored soon after.
* A page answer carries only the start of a page. A caller that wants more must ask again
  with a larger limit.
* A page call cannot read a page fresher than the age an operator configures.
* A search asks SearXNG for its whole answer whatever number of results the caller wants,
  so a small number saves the caller reading, not SearXNG searching.
* The corpus holds only the newest markdown of a page. A caller that asks for a version the
  corpus no longer holds cannot read that version again.
