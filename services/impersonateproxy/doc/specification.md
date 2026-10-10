# impersonateproxy — Technical Specification

## Context

`impersonateproxy` is a forward HTTP proxy that fetches each page as Chrome
fetches it. Some origins refuse or change pages for clients that do not look
like a browser.

## Glossary

* **Fingerprint** — the TLS ClientHello, the HTTP/2 SETTINGS, window and
  pseudo-header order, and the header fields and their order.

## Non-Goals

* Running JavaScript or rendering pages.
* Caching responses.
* Terminating client TLS or relaying `CONNECT` tunnels for clients.
* Enforcing target-safety; that stays with the egress proxy.

## Functional Requirements

* The service SHALL accept `GET` and `HEAD` requests with an absolute `http`
  or `https` target.
* The service SHALL answer every other method, `CONNECT` too, with `405`, and
  another target with `400`, contacting no origin.
* The service SHALL send each request through the configured egress proxy in
  a `CONNECT` tunnel, and SHALL fail startup when no egress proxy is set.
* The service SHALL send each request with the fingerprint of the Chrome
  version that the startup log names.
* The service SHALL send the header fields of Chrome in the order of Chrome,
  and no `sec-` fields and no `br` or `zstd` coding for an `http` target.
* The service SHALL pass on the client's `Cookie`, `Referer`,
  `If-None-Match` and `If-Modified-Since`, and no other client header.
* The service SHALL send the default referer when the client sends no
  `Referer`.
* The service SHALL relay the origin's status, end-to-end headers and body,
  and SHALL NOT follow a redirect.
* The service SHALL decode a body in `gzip`, `deflate`, `br` or `zstd`, and
  SHALL then remove `Content-Encoding` and `Content-Length`.
* The service SHALL relay a body in another coding as it gets it.
* The service SHALL answer `502` when the egress proxy or the origin fails
  before the response headers.
* The service SHALL end the response early when the body fails after the
  response headers.

## Non-Functional Requirements

* Operational behavior SHALL be observable through Prometheus metrics on the
  ops address.

## Known Limitations

* The header fields for an `https` target are those that Chrome sends over
  HTTP/2, also when the origin speaks HTTP/1.1.
* The service sends `Sec-Fetch-Site: cross-site` for a referer of another
  origin on the same site.
* A newer Chrome version needs a newer tls-client release.
