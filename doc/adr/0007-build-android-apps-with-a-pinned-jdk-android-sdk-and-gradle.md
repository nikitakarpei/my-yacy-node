# 7. Build Android apps with a pinned JDK, Android SDK and Gradle

Date: 2026-11-06

## Status

Accepted

## Context

Services of the stack must also run on Android phones that operators own. An Android app
needs a JDK, the Android SDK, an Android Gradle Plugin build, and Kotlin for its code.
Peers must be able to build the same APK from the same commit.

## Decision

- `tools/tools.lock` pins the Temurin JDK and each Android SDK package (platform, build
  tools, platform tools and NDK) by the SHA-256 of its archive from Google's repository.
  `tools/install` unpacks each archive into `.toolchain/android-sdk`, without sdkmanager.
- Each app is a Gradle project in `distributions/<service>android`. The Gradle build in
  `libraries/androidservicehost` holds the convention plugin of the apps, their shared
  Kotlin code and the Gradle wrapper. It pins the Android Gradle Plugin and Kotlin.
- `gradle/verification-metadata.xml` of each app pins each file that the build downloads.
- `tools/install android` installs this toolchain. `make android-apps` builds, lints and checks the apps,
  and `make android-apps-reproducible` checks that two builds of one commit give the same APKs.
  `make verify` does not build the apps.

## Consequences

- No sdkmanager accepts the SDK licenses. The operator who builds accepts them.
- Google publishes no Android build tools for Linux on ARM, so these hosts cannot build
  the apps.
- A new SDK version needs new checksums in `tools/tools.lock`. The F-Droid SDK
  transparency log has the checksums of the Linux archives.
- The toolchain adds some GB to `.toolchain`.
