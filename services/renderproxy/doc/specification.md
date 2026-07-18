# renderproxy — Technical Specification

## Context

`renderproxy` is a forward HTTP proxy that serves a requested URL by driving an external
CDP-compatible browser to load it, returning the loaded page in place of the raw response
an origin would give. It exists so an HTTP client can stay a plain fetcher and still
obtain script-loaded pages.

It accepts proxy requests in absolute-URL form only. It contacts the requested origin
itself: a redirect response is relayed to the client unchanged; any other response is
rendered by driving the browser and returned. The service's own origin requests leave
through a required egress proxy; the browser reaches origins through its own configured
proxy.

## Non-Goals

* Terminating origin TLS or serving HTTP CONNECT tunnels.
* Producing, storing, or forwarding pages to any consumer of its own.
* Deciding what to fetch, recrawl, rank, or index.
* Enforcing target-safety, host politeness, or robots policy; those stay with the egress and
  browser proxies.
* Running or supervising the browser process itself.
* Defeating anti-bot walls; a wall is a refusal to relay, not an obstacle to evade.

## Functional Requirements

* The service SHALL answer a proxied GET whose origin returns a redirect with that redirect's
  status and `Location`, and SHALL not drive the browser for it.
* The service SHALL return, for a proxied GET whose origin does not redirect, the target page
  as the browser has it once loading settles.
* The service SHALL route its own origin requests through the configured egress proxy, and
  SHALL fail startup when no egress proxy is configured.
* The service SHALL relay the origin's HTTP status and refusal responses to the client
  unchanged.
* The service SHALL answer an HTTP CONNECT request with a refusal status, contacting no
  origin.
* The service SHALL answer a request that carries no absolute target URL with a client-error
  status, contacting no origin.
* The service SHALL fail a request with an error status when the browser is unreachable or
  the page does not settle within the request deadline.

## Non-Functional Requirements

* Every request SHALL be bound by an explicit deadline and an explicit response-size cap.
* Concurrent renders SHALL be capped, and requests past the cap SHALL wait rather than
  overcommit the browser.
* The browser SHALL be a separate CDP endpoint reached over a narrow interface, replaceable
  between Chrome, Playwright, or any CDP peer with no change to request handling.
* Operational behavior SHALL be observable through machine-readable metrics.

## Known Limitations

* Because CONNECT is refused, a client reaches the service in absolute-URL proxy mode only;
  a client that can only tunnel cannot use it.
* Latency, memory, and fidelity follow the attached browser; a heavy page costs far more
  than a raw fetch.
* Egress filtering, target-safety, and host politeness hold only as far as the service's
  egress proxy and the browser's own proxy enforce them; the service adds none.
* The browser's own TLS trust and version govern which origins load; the service adds no
  certificates of its own.
