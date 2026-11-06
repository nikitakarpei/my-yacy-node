# 11. Read pairing codes in the Android app with ZXing

Date: 2026-11-07

## Status

Accepted

## Context

The companion app pairs with a peer from the QR code that the peer shows. The phone must read
the code in the app, also on phones without Google Play services, such as GrapheneOS.

## Decision

ZXing core 3.5.4 decodes the QR codes in the camera pictures. The app takes the pictures with
the Camera2 API of the platform, so no camera library is necessary.

## Consequences

- The app carries ZXing core, a library in maintenance mode.
- The app has its own camera screen.
