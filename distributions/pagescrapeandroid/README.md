# pagescrape for Android

This app runs [pagescrape](../../services/pagescrape/doc/specification.md) on an Android
phone, in the app process.

## Build

Run these commands at the root of the repository:

```sh
./tools/install
. .toolchain/activate
make android-apps
```

`./tools/install` installs the pinned tools, the JDK, the Android SDK, the NDK and Gradle
into `.toolchain`. `make android-apps` builds, lints and checks the app. The unsigned release APK is
`build/outputs/apk/release/pagescrapeandroid-release-unsigned.apk`.

Two clean builds of one commit give the same APK, byte for byte. `make android-apps-reproducible`
checks this. Gradle checks each file that the build downloads against
`gradle/verification-metadata.xml`. After you change a build dependency, run
`make android-apps-verification-metadata` and examine the change to this file.

## Install

Sign the APK with your own key. Then install it with `adb`:

```sh
. .toolchain/activate
"$ANDROID_HOME"/build-tools/*/apksigner sign --ks release.jks --out pagescrape.apk \
  distributions/pagescrapeandroid/build/outputs/apk/release/pagescrapeandroid-release-unsigned.apk
"$ANDROID_HOME"/platform-tools/adb install pagescrape.apk
```

## Use

Open the app. Fill in the fields of the environment that have no default, change others
if you need to, then tap **Save and start**. The app keeps the values for the next start. The app keeps the service running in a foreground service.
Allow the app to run in the background, or the phone can stop the service while the
screen is off. Tap **Stop** to stop the service.
