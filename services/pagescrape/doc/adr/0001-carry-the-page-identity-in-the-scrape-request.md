# 1. Carry the page identity in the scrape request

Date: 2026-08-27

## Status

Accepted. ADR 2 supersedes its redirect identity rule.

## Context

A live page usually has one URL for its identity and its content. An archived
or mirrored page has a public identity but supplies its content from another
URL. One request value cannot preserve both facts.

## Decision

A scrape request carries the page URL that identifies the result and can carry
a separate fetch URL. When the fetch URL is absent, the service reads the page
URL.

The request producer is responsible for the relationship between the two URLs.

## Considered alternatives

Deriving the page identity from the fetched content is rejected because the
content can omit or misstate it.

Translating page URLs inside the scrape service is rejected because it would
bind the service to one content source.

## Consequences

Live, archived, mirrored, and cached pages use one request contract. The scrape
service does not need source-specific rules.

The service cannot verify that a fetch URL supplies the page that the page URL
names.
