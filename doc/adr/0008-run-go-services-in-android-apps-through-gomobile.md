# 8. Run Go services in Android apps through gomobile

Date: 2026-11-06

## Status

Accepted

## Context

An Android app is Kotlin or Java code. The services of the stack are Go code. The app
must run a service as it is, in the app process.

## Decision

`golang.org/x/mobile`, pinned as Go tool directives in the Go module of the app, binds the
Go packages of the app into one Android library with `gomobile bind`.

- The app starts each service through its `assembly`, the same way its `cmd` does, from
  environment variables that the app sets.
- One library holds every package, because each library carries the Go runtime.
- The library holds code for 64-bit ARM only.
- The build drops the Go module information and the Go build ID. gomobile writes absolute
  paths into them, so they would change the APK with the path of the checkout.
- gomobile sends the standard error of Go to logcat, so the service logs go there.

## Consequences

The Go types that cross into Kotlin are limited to the types that gomobile supports. The
app carries the Go runtime and its services, some tens of MB.
