# YaCy Companion — Technical Specification

## Context

Peers of the YaCy network cannot fetch again each page that another peer indexed, so they cannot
check that its index is honest. The app lets a device that the operator owns fetch pages for the
peer and build their RWIs, with signatures by a key in the secure hardware of the device, so that
other peers can check what the device did.

## Non-Goals

* Checks of the device with Play Integrity, or signing of the APK by Google Play.

## Functional Requirements

* Before pairing, the app SHALL show what it is, the permissions that it needs, and the data
  that it shares.
* The app SHALL NOT start pairing before the user allows notifications and running in the
  background, and agrees to the data sharing.
* The app SHALL ask for the camera only to read the QR code of a pair link.
* The app SHALL pair with the peer whose pair link it reads with the camera.
* Pairing SHALL succeed only when the peer accepts the device.
* When pairing fails, the app SHALL tell why.
* The app SHALL connect to the broker of the peer only when the broker has the key that the
  pair link names.
* When the peer no longer accepts the device, the app SHALL stop its work and show why, and SHALL
  stay open until the user closes it.
* When the app and the peer speak different protocols, the app SHALL stop its work and tell which
  of the two to update.
* When the device no longer has the key that it paired with, the app SHALL stop its work and show
  why.
* Until the user stops it or unpairs the device, the paired app SHALL fetch pages and build RWIs
  when the rules below allow it.
* The app SHALL sign each page that it fetches with a hardware-attested key of the device.
* The app SHALL NOT create or change the streams and consumers of the peer.
* The app SHALL show whether it is connecting, connected, waiting for a charger, for Battery Saver
  to turn off, or for an unmetered network, or disconnected, and why it is disconnected.
* Unless the user allows mobile data, the app SHALL fetch pages only over unmetered networks.
* Unless the user turns off "Run only when charging", the app SHALL fetch pages only while the
  device is on a charger.
* Unless the user turns off "Pause while Battery Saver is on", the app SHALL NOT fetch pages
  while Battery Saver is on.
* The app SHALL notify the user once when it loses the peer, and remove the notice when it
  connects again.
* Stop in the notification of the app SHALL stop its work and close the app.

## Non-Functional Requirements

* When disconnected, the app SHALL try again after a pause that doubles from 5 s to at most 60 s.
* When the device gets, loses, or validates a network, VPNs included, the app SHALL try again at
  once and start the pause again from 5 s.
* Without a network, the app SHALL wait until the device has one.
* An RWI that the app builds SHALL be the same as the RWI that pagerwi builds from the same page.
* Two builds of one commit SHALL give the same unsigned APK, byte for byte.
* The app SHALL fetch pages and connect to the peer only with TLS.
* The app SHALL NOT put its data in backups or in transfers to another device.
* Other apps SHALL NOT start or call the app, except to open its home screen.
* The app SHALL NOT show web pages, and SHALL open each link in the browser of the device.
* A release build SHALL NOT let a debugger connect.

## Known Limitations

* The app runs only on 64-bit ARM devices with Android 14 or later.
* The release APK is not signed. The operator signs it with an own key.
* The peer does not check the key attestation of the device yet.
* pagescrape and pagerwi do not sign their page offers and RWIs yet.
* The app writes the logs of its fetches and RWIs to logcat, under the tag `GoLog`.
* A debug build cannot sign, so each page fetch of a debug build fails.
* Each start of the app after a stop creates a new signing key.
* The app holds a wake lock while it pairs, connects to the peer, works for it, or waits to
  connect again, so the device uses more power.
* Two builds give the same APK only when they use the toolchain that `tools/install android`
  installs.
