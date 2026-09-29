# 2. Read visible text without a main-text extractor

Date: 2026-09-29

## Status

Accepted

## Context

Recipe 4 read the main text of a page with trafilatura. Go has no exact
port of trafilatura. A cross-validated retrain on the judged pages gave
the same average precision with the visible text of the page as with
go-trafilatura or go-readability.

## Decision

- Recipe 5 reads the title and the visible text of the page.
- The library uses no main-text extractor.

## Consequences

The library has fewer dependencies. Text in menus and footers goes into the
text family.
