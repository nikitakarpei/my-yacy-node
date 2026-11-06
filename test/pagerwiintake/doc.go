// Package e2e runs the end-to-end test of the RWI intake of a page across containers.
//
// It starts NATS JetStream, a static origin page, an egress proxy, the scrape service, the
// pagerwi service, and a node on a hermetic Docker network. It then publishes one scrape
// request from the host and asks the node how many URLs it stores. The page that the scrape
// service offers reaches the node as its RWI.
//
// The test is guarded by the e2e build tag and needs a working Docker daemon.
// Run it with `make e2e`. It is not part of the `make verify` quality gate.
package e2e
