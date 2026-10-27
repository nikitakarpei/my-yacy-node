# 5. Draw the dashboards with uPlot

Status: accepted

## Context

The dashboard page draws time series in the browser. The page has no build step, and the
binary carries every file that the page loads.

uPlot is a small chart library for time series, under the MIT licence.

## Decision

Embed uPlot 1.6.32 in `yacypeer`, as its published files, next to the uPlot licence.

## Consequences

* The page needs no network access beyond the peer.
* An update of uPlot replaces the embedded files.
