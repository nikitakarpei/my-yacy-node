// Package e2e runs the end-to-end recall test against containers.
//
// It starts NATS JetStream, a static origin page, an egress proxy, yacycrawler
// (with its markdown representation enabled), corpusmarkdown, and corpusrecall on
// a hermetic Docker network, then calls the corpusrecall gRPC port from the host.
// The recall triggers an on-demand crawl, waits for the markdown to be stored,
// and returns it, while reporting the unserved text representation as unavailable.
//
// The test is guarded by the e2e build tag and needs a working Docker daemon.
// Run it with `make e2e-corpusrecall`. It is not part of the `make verify` gate.
package e2e
