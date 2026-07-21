// Package e2e runs the end-to-end markdown-storage test against containers.
//
// It starts NATS JetStream, a static origin page, an egress proxy, yacycrawler
// (with its markdown representation enabled), and corpusmarkdown on a hermetic
// Docker network, then drives the crawler over NATS from the host and reads the
// stored markdown back from the JetStream Object Store by its URL-addressed name.
//
// The test is guarded by the e2e build tag and needs a working Docker daemon.
// Run it with `make e2e`. It is not part of the `make verify` quality gate.
package e2e
