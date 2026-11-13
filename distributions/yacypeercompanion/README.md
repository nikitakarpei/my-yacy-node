# YaCy Companion

YaCy Companion turns an Android device into a trusted helper of your YaCy peer. The device fetches
web pages for the peer and signs each one with a key in its secure hardware, so other peers can
check that the pages are genuine.

## Install

The release workflow of this repository builds, signs and publishes each release of the app.
It runs for a tag `yacypeercompanion-v<version>` and signs each release with a new key. Each
release holds the signed APK, the unsigned APK, the release record `release.json` and its
Sigstore bundle. No release is published yet.

Check a downloaded APK before you install it. Use these commands in a clone of this repository:

```sh
gh release verify-asset yacypeercompanion-v0.1.0 yacypeercompanion-v0.1.0.apk
gh attestation verify yacypeercompanion-v0.1.0.apk -R nikitakarpei/my-yacy-node
"$ANDROID_HOME"/platform-tools/adb install yacypeercompanion-v0.1.0.apk
```

The app runs on 64-bit ARM devices with Android 14 or later. Android does not update an app
that another key signed, so to update, uninstall the app, install the new release, and pair
again.

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
`.toolchain`. `make android-app` builds the app, runs its tests, lints it, and checks the
attack surface of the release APK. The unsigned
release APK is `app/build/outputs/apk/release/app-release-unsigned.apk`. Sign it with an own key
and install it:

```sh
"$ANDROID_HOME"/build-tools/*/apksigner sign --ks release.jks --out yacypeercompanion.apk \
  distributions/yacypeercompanion/app/build/outputs/apk/release/app-release-unsigned.apk
"$ANDROID_HOME"/platform-tools/adb install yacypeercompanion.apk
```

Two clean builds of one commit give the same APK, byte for byte. `make android-app-reproducible`
checks this. Gradle checks each file that the build downloads against
`gradle/verification-metadata.xml`. After you change a build dependency, run
`make android-app-verification-metadata` and examine the change to this file.

`make android-app-vulnerabilities` finds known vulnerabilities in the libraries of the release
APK, and `make vulnerabilities-go` in the Go modules. A CI workflow runs both each week.
