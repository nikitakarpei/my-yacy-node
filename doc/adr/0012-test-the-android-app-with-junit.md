# 12. Test the Android app with JUnit

Date: 2026-11-07

## Status

Accepted

## Context

The state of the companion app changes in one transition function in plain Kotlin. Its tests
run on the JVM of the build machine and need a test framework.

## Decision

JUnit 4.13.2 runs the unit tests of the app. `make android-app` runs them.

## Consequences

- JUnit is a test dependency only. The APK does not carry it.
