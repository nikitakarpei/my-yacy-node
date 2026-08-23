// Package e2e runs the end-to-end fan-out test across containers.
//
// It starts NATS JetStream, provisions the scrape-request stream the way an operator
// does, and starts a static origin page, an egress proxy, Manticore, corpusmarkdown, and
// corpustext on a hermetic Docker network. It then publishes one scrape request from the
// host and reads both corpora: the markdown object out of the JetStream Object Store and
// the indexed text out of Manticore.
//
// The test is guarded by the e2e build tag and needs a working Docker daemon.
// Run it with `make e2e`. It is not part of the `make verify` quality gate.
package e2e
