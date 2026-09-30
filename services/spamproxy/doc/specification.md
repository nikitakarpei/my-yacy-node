# spamproxy — Technical Specification

## Context

`spamproxy` is a forward HTTP proxy that adds a spam verdict to the HTML pages it relays. It
exists so that each consumer can apply its own spam policy without scoring pages itself.

## Glossary

* **Page** — a `2xx` response to `GET` with a `text/html` or `application/xhtml+xml` body.
* **Verdict** — `spam` when the page score is above the model threshold, else `clean`.

## Non-Goals

* Blocking, rewriting, or ranking pages.
* Caching responses.
* Terminating client TLS.
* Enforcing target-safety, host politeness, or robots policy; those stay with the egress
  proxy.

## Functional Requirements

* The service SHALL accept `GET` and `HEAD` requests with an absolute `http` or `https`
  target.
* The service SHALL answer every other method with `405`, and a relative target or another
  scheme with `400`, contacting no origin.
* The service SHALL send each request through the configured egress proxy, and SHALL fail
  startup when no egress proxy is configured.
* The service SHALL send an `http` request in absolute-URL form, and an `https` request
  through a `CONNECT` tunnel or in absolute-URL form, as the dial mode selects.
* The service SHALL pass on the client's `User-Agent`, `If-None-Match`, `If-Modified-Since`,
  and `Accept-Encoding` request headers, and no other request header.
* The service SHALL remove from `Accept-Encoding` each encoding that it cannot decode, and
  SHALL send `identity` when none remains.
* The service SHALL relay the origin's status, end-to-end headers, and body, and SHALL NOT
  follow a redirect.
* The service SHALL add to each page one `Spam-Assessment` header, an RFC 9651 Item such as
  `spam;score=0.935;threshold=0.8;model="2026-09"`.
* The header SHALL carry the verdict as its token, the score to three decimals, the model
  threshold, and the model file version.
* The service SHALL remove every `Spam-Assessment` header that the origin sends, so a
  response that is not a page has no such header.
* The service SHALL assess a page larger than the page byte ceiling on its first bytes up to
  the ceiling.
* The service SHALL assess a `gzip` or `deflate` page on its decoded body.
* The service SHALL answer `502` for a page in another encoding, such as `br`, or with a body
  that does not decode.
* The service SHALL answer `503` with `Retry-After` when a page waits for the page limit or
  for a free assessment until the response header timeout ends, or when the assessment of a
  page fails.
* The service SHALL answer `502` when the egress proxy fails, or stops before the service has
  read the part of a page that it assesses.
* The service SHALL answer `504` when the upstream response, the part of a page that it
  assesses, or the assessment does not end within the response header timeout after the
  request arrives.
* The service SHALL use the `wait` of a `Prefer` header as the response header timeout when
  the `wait` is shorter.
* The service SHALL fail startup when the recipe version of the model file is not the recipe
  version of the `spammodel` library.

## Non-Functional Requirements

* The service SHALL compute features and scores only through the `spammodel` library, the
  same code that computes features to train the spam model.
* The service SHALL hold in memory only the part of a page that it assesses, and SHALL
  stream the rest through.
* Operational behavior SHALL be observable through Prometheus metrics on the ops address.

## Known Limitations

* The service reads the model file at startup only; a new model file needs a restart.
* A cache in front of the service keeps old verdicts until their copies expire.
* An HTML body with another media type, such as `text/plain`, gets no header.
