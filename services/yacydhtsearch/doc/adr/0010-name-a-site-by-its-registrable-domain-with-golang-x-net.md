# 10. Name a site by its registrable domain with golang.org/x/net

Date: 2026-10-07

## Status

Accepted

## Context

The named site entry score rewards a site whose name holds the query words. The
site name holds every label of the host, so `linux.buzzing.cc` and
`windows.apkpure.com` score as if they were the site of the query. For the query
`windows`, such download pages took the first ten.

A subdomain that holds the query word is sometimes a real section about it, such
as `linux.developpez.com`. The address alone cannot tell the two apart.

The standard library has no public suffix list. `golang.org/x/net` is already in
`go.mod`, through `documentextraction`.

## Decision

- `documentrelevance` finds the registrable domain of a host with
  `golang.org/x/net/publicsuffix`. It uses only the ICANN part of the list, so a
  host under a hosting suffix such as `debian.net` keeps the name `debian`.
- A site whose registrable domain holds no query word keeps a fifth of its named
  site entry score.

## Consequences

- On the judged queries, the mean gain over the queries of several relevant
  documents rises from 0.7234 to 0.7290. `windows` rises from 0.066 to 0.294.
- A subdomain that names a real section keeps only a fifth of its score.
- The public suffix list changes only with a new version of `golang.org/x/net`.
