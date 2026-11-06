# pagerwi for Android — Technical Specification

## Context

The app runs [pagerwi](../../../services/pagerwi/doc/specification.md) in the
process of an Android app. The service is the same code that its container image runs.
The operator gives the service its environment variables in the fields of the app.

## Non-Goals

* Attesting what the service builds or publishes.
* Finding the brokers without configuration.
* Showing the logs of the service on the screen.

## Functional Requirements

* The app SHALL show one field for each environment variable that the service reads,
  with the default of the variable.
* The app SHALL run the service with the values that the operator saved last, and
  with the default of each variable whose field is empty.
* The app SHALL keep the saved values when it stops and starts again.
* The app SHALL keep the service running while the screen is off, when the operator
  allows the app to run in the background.
* The app SHALL show whether the service runs, and why it stopped.
* The app SHALL stop the service when the operator stops it.
* The app SHALL notify the operator when the service stops without a request of the
  operator, and tell why it stopped.
* The app SHALL NOT start the service again while it runs.
* Two builds of one commit SHALL give the same unsigned APK, byte for byte.

## Known Limitations

* The app runs only on 64-bit ARM phones with Android 9 or later.
* The release APK is not signed. The operator signs it with an own key.
* The service writes its logs to logcat, under the tag `GoLog`.
* The app cannot notify the operator when Android stops the process of the app.
* The app holds a wake lock while the service runs, so the phone uses more power.
* The app saves the environment unencrypted in the storage of the app.
* Two builds give the same APK only when they use the toolchain that `tools/install android`
  installs.
