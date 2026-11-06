# 10. Time-stamp statements in the Android app with Bouncy Castle

Date: 2026-11-07

## Status

Accepted

## Context

Each statement that the app signs gets an RFC 3161 time-stamp, and each signature must use the
low-S form that the Hardware-Attested Statements format requires. The Android platform has no
API for time-stamp requests and responses, and none for DER signature values.

## Decision

Bouncy Castle 1.86 (`bcpkix-jdk18on`) builds the time-stamp requests, checks the responses
against the pinned certificate of the time-stamp authority, and encodes the signatures. The
app does not register it as a security provider, so it cannot replace the crypto of Android.

## Consequences

- The app carries Bouncy Castle.
- Android lint reports a trust-all trust manager in its EST client, which the app does not use.
  `lint.xml` of the app ignores this finding for the Bouncy Castle archive only.
