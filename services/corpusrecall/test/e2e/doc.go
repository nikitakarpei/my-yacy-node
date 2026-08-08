// Package e2e runs the end-to-end recall tests against containers.
//
// Each test starts the crawl and corpus services on a hermetic Docker network, then
// calls the corpusrecall gRPC port from the host and checks the recalled page.
//
// The tests are guarded by the e2e build tag and need a working Docker daemon.
// Run them with `make e2e-corpusrecall`. They are not part of the `make verify` gate.
package e2e
