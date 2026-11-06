# 8. Run Go services in Android apps through gomobile

Date: 2026-11-06

## Status

Accepted

## Context

An Android app is Kotlin or Java code. The services of the stack are Go code. The app
must run a service as it is, in the app process.

## Decision

`golang.org/x/mobile`, pinned as Go tool directives in the Go module of each app, binds one
Go package of the app into an Android library with `gomobile bind`.

- Each bound package starts the service through its `assembly`, the same way its
  `cmd` does, from environment variables that the app gives as text.
- The library holds code for 64-bit ARM only.
- The build drops the Go module information and the Go build ID. gomobile writes absolute
  paths into them, so they would change the APK with the path of the checkout.
- gomobile sends the standard error of Go to logcat, so the service logs go there.

## Consequences

The Go types that cross into Kotlin are limited to the types that gomobile supports. Each
app carries the Go runtime and its service, some tens of MB.
