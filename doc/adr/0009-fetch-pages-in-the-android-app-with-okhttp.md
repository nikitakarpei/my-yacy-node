# 9. Fetch pages in the Android app with OkHttp

Date: 2026-11-07

## Status

Accepted

## Context

The capture proxy of the companion app records each exchange as it went over the wire: the
request with all its headers, the address it connected to, and the TLS handshake. The HTTP
client of the Android platform does not expose these facts.

## Decision

OkHttp 5.5.1 fetches the pages, the drand beacons and the time-stamps of the capture proxy. It
is the version that attested-web-capture-android uses, which the capture code comes from.

## Consequences

- The app carries OkHttp and Okio.
- An update of OkHttp needs new entries in the verification metadata of the app.
