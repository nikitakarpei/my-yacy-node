# classifierproxy — Technical Specification

## Context

`classifierproxy` is a forward HTTP proxy that adds a classification to the HTML pages it
relays: one verdict for each category. Each consumer can then apply its own policy for each
category without scoring pages itself.

## Glossary

* **Page** — a `2xx` response to `GET` with a `text/html` or `application/xhtml+xml` body.
* **Category** — a kind of page, such as `spam`, that the operator gives one model for.
* **Verdict** — true when the page score of a category is above the threshold of its model,
  else false.

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
* The service SHALL add to each page one `Page-Classification` header, an RFC 9651
  Dictionary with one member for each category, such as
  `spam;score=0.935;threshold=0.800;model="2026-09", adult=?0;score=0.007;threshold=0.800;model="2026-11"`.
* The key of a member SHALL be the category, and its value SHALL be the verdict as a
  Boolean. The parameters SHALL be the score and the model threshold to three decimals, and
  the model file version.
* The members SHALL come in the order of their scores, the highest first, then of their
  categories.
* The service SHALL remove every `Page-Classification` header that the origin sends, so a
  response that is not a page has no such header.
* The service SHALL classify a page larger than the page byte ceiling on its first bytes up
  to the ceiling.
* The service SHALL classify a `gzip` or `deflate` page on its decoded body.
* The service SHALL answer `502` for a page in another encoding, such as `br`, or with a body
  that does not decode.
* The service SHALL answer `503` with `Retry-After` when a page waits for the page limit or
  for a free classification until the response header timeout ends, or when the
  classification of a page fails.
* The service SHALL answer `502` when the egress proxy fails, or stops before the service has
  read the part of a page that it classifies.
* The service SHALL answer `504` when the upstream response, the part of a page that it
  classifies, or the classification does not end within the response header timeout after
  the request arrives.
* The service SHALL use the `wait` of a `Prefer` header as the response header timeout when
  the `wait` is shorter.
* The service SHALL fail startup when the recipe version of a model file is not the recipe
  version of the `classifiermodel` library.

## Non-Functional Requirements

* The service SHALL compute features and scores only through the `classifiermodel` library,
  the same code that computes features to train the models.
* The service SHALL hold in memory only the part of a page that it classifies, and SHALL
  stream the rest through.
* Operational behavior SHALL be observable through Prometheus metrics on the ops address.

## Known Limitations

* The service reads the model files at startup only; a new model file needs a restart.
* A cache in front of the service keeps old classifications until their copies expire.
* An HTML body with another media type, such as `text/plain`, gets no header.
