// Package e2e runs the end-to-end full-text indexing test against containers.
//
// It starts NATS JetStream, a static origin page, an egress proxy, a search index,
// yacynode, yacycrawler, and corpustext on a hermetic Docker network, then drives
// the crawler over NATS from the host. It polls the search index's own search API
// and the corpustext search for the indexed page. It runs once for Elasticsearch
// and once for Manticore.
//
// The test is guarded by the e2e build tag and needs a working Docker daemon.
// Run it with `make e2e`. It is not part of the `make verify` quality gate.
package e2e
