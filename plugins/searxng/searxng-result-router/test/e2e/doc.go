// Package e2e runs the end-to-end result-link-router test against containers.
//
// It starts a NATS JetStream broker, the visitcrawl service, the real
// searxng/searxng image with the plugin and a self-contained test engine
// mounted in, and one reverse proxy serving both of them as a single
// results origin. It then drives a search from the host: it checks that
// the returned result link routes through /visit on that origin, that
// following that link redirects to the original destination, and that
// doing so places a crawl order on NATS.
//
// The test is guarded by the e2e build tag and needs a working Docker
// daemon. Run it with `make e2e-plugin`. It is not part of the `make verify`
// quality gate.
package e2e
