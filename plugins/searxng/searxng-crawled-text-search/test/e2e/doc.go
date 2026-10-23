// Package e2e runs the end-to-end crawled-text-search test against containers.
//
// It starts three origin pages behind an egress proxy, corpustext, and the
// search index server corpustext writes to when the run names one, publishes one scrape request per origin, and starts the real
// searxng/searxng image with the engine module mounted in, then drives a search
// from the host through corpustext and checks that the returned result carries
// the indexed title, URL, and matched content. It runs once for each search
// index corpustext can write to.
//
// The test is guarded by the e2e build tag and needs a working Docker
// daemon. Run it with `make e2e-searxng-crawled-text-search`. It is not
// part of the `make verify` quality gate.
package e2e
