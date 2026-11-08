# YaCy Companion

YaCy Companion turns an Android device into a trusted helper of your YaCy peer. The device fetches
web pages for the peer and signs each one with a key in its secure hardware, so other peers can
check that the pages are genuine.

## Install

No release of the app is published yet. Build it from source, as below, then sign the APK with
your own key and install it with `adb`:

```sh
. .toolchain/activate
"$ANDROID_HOME"/build-tools/*/apksigner sign --ks release.jks --out yacypeercompanion.apk \
  distributions/yacypeercompanion/app/build/outputs/apk/release/app-release-unsigned.apk
"$ANDROID_HOME"/platform-tools/adb install yacypeercompanion.apk
```

The app runs on 64-bit ARM devices with Android 14 or later.

## Use

Open the app and follow its screens: allow the permissions, agree to the data sharing, then
scan the QR code from the Companions tab in the web UI of your peer.

## Build from source

Run these commands at the root of the repository:

```sh
./tools/install
. .toolchain/activate
make android-app
```

`./tools/install` installs the pinned tools, the JDK, the Android SDK, the NDK and Gradle into
`.toolchain`. `make android-app` builds the app, runs its tests, and lints it. The unsigned
release APK is `app/build/outputs/apk/release/app-release-unsigned.apk`.

Two clean builds of one commit give the same APK, byte for byte. `make android-app-reproducible`
checks this. Gradle checks each file that the build downloads against
`gradle/verification-metadata.xml`. After you change a build dependency, run
`make android-app-verification-metadata` and examine the change to this file.
