# 13. Draw pairing codes in yacypeer with rsc.io/qr

Date: 2026-11-07

## Status

Accepted

## Context

The web UI of yacypeer shows a QR code that the companion app scans to pair. The peer must
draw the code itself, because the web UI loads no scripts or images from other sites.

## Decision

rsc.io/qr 0.2.0 encodes the pairing link. The peer draws the modules of the code as SVG in
the page.

## Consequences

- yacypeer carries rsc.io/qr, a small library without dependencies that gets no changes.
- The library always uses the same mask, so a code can be a little harder to scan than one
  with the best mask.
