// Package e2e runs the end-to-end web-research test against containers.
//
// It starts NATS JetStream, a static origin page, an egress proxy, corpusmarkdown,
// the real searxng/searxng image with the result-link-router plugin and a
// self-contained test engine mounted in, and webresearchmcp on a hermetic Docker
// network. It then drives the service from the host over the Model Context
// Protocol: it lists the tools, searches, and reads one page as markdown.
//
// The test is guarded by the e2e build tag and needs a working Docker daemon.
// Run it with `make e2e-webresearchmcp`. It is not part of the `make verify`
// quality gate.
package e2e
