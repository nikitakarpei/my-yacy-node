# YaCy Companion — Technical Specification

## Context

Peers of the YaCy network cannot fetch again each page that another peer indexed, so they cannot
check that its index is honest. The app lets a phone that the operator owns fetch pages for the
peer and build their RWIs, with signatures by a key in the secure hardware of the phone, so that
other peers can check what the phone did.

## Non-Goals

* Checks of the phone with Play Integrity, or signing of the APK by Google Play.
* Authentication with the broker of the peer.

## Functional Requirements

* Before pairing, the app SHALL show what it is and the permissions that it needs.
* The app SHALL NOT start pairing before the user allows notifications and running in the
  background.
* The app SHALL ask for the camera only to read a pairing code.
* The app SHALL pair with the peer whose pairing code it reads with the camera.
* Pairing SHALL succeed only when the phone is ready to take scrape requests and to deliver RWIs
  over the broker of the peer.
* When pairing fails, the app SHALL tell why.
* After pairing, the app SHALL fetch pages and build RWIs until the user stops it or unpairs the
  phone.
* The app SHALL sign each page that it fetches with a hardware-attested key of the phone.
* The app SHALL show whether it is connecting, connected, or disconnected, and why it is
  disconnected.
* Unless the user allows mobile data, the app SHALL fetch pages only over unmetered networks.
* The app SHALL notify the user once when it loses the peer, and remove the notice when it
  connects again.
* Stop in the notification of the app SHALL stop its work and close the app.

## Non-Functional Requirements

* When disconnected, the app SHALL try again after a pause that doubles from 5 s to at most 60 s.
* When the phone gets, loses, or validates a network, VPNs included, the app SHALL try again at
  once and start the pause again from 5 s.
* Without a network, the app SHALL wait until the phone has one.
* An RWI that the app builds SHALL be the same as the RWI that pagerwi builds from the same page.
* Two builds of one commit SHALL give the same unsigned APK, byte for byte.

## Known Limitations

* The app runs only on 64-bit ARM phones with Android 9 or later.
* The release APK is not signed. The operator signs it with an own key.
* The peer does not check yet that it accepts the signatures of the phone.
* pagescrape and pagerwi do not sign their page offers and RWIs yet.
* The app writes the logs of its fetches and RWIs to logcat, under the tag `GoLog`.
* A debug build cannot sign, so each page fetch of a debug build fails.
* Each start of the app after a stop creates a new signing key.
* The app uses the default stream limits. A peer with other limits gets its streams changed.
* The app holds a wake lock while it runs, so the phone uses more power.
* Two builds give the same APK only when they use the toolchain that `tools/install android`
  installs.
