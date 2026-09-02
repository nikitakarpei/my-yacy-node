// Package e2e runs the end-to-end scrape test against containers.
//
// It starts NATS JetStream, two origins, an egress proxy, and pagescrape on a hermetic
// Docker network, then publishes a scrape request from the host and reads what the service
// offers the corpora: the page when the origin serves it, a failure when it does not.
//
// The test is guarded by the e2e build tag and needs a working Docker daemon.
// Run it with `make e2e-pagescrape`. It is not part of the `make verify` quality gate.
package e2e
