# 14. Hold the Android app to the MASVS L1, L2 and privacy controls

Date: 2026-11-09

## Status

Accepted

## Context

The companion app keeps a hardware key that signs the pages that it fetches, and it connects to
the broker of a peer. The OWASP Mobile Application Security Verification Standard (MASVS) 2.1.0
gives the security and privacy controls of mobile apps. The tests of the OWASP Mobile
Application Security Testing Guide (MASTG) check these controls for the profiles L1, L2 and R.

## Decision

The app meets the MASVS controls at the profiles L1 and L2, and the MASVS-PRIVACY controls.

- The key attestation that the peer checks shows the integrity of the device and of the app
  (MASVS-RESILIENCE-1 and -2).
- The app does not hide its code or stop debuggers (MASVS-RESILIENCE-3 and -4), because its
  build is open and reproducible.
- MASVS-AUTH-2 and -3 do not apply, because the app has no user login.

## Consequences

Checks in the build and in CI must keep each control that a change to the app can break. The
integrity controls depend on the check of the key attestation by the peer, which the peer does
not do yet.
