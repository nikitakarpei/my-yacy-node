// Package e2e runs the end-to-end scrape test against containers.
//
// It starts NATS JetStream, a static origin page, an egress proxy, yacycrawler,
// corpusmarkdown, corpusrecall, and firecrawlshim on a hermetic Docker network,
// then calls the firecrawlshim HTTP /v1/scrape endpoint from the host. The scrape
// triggers an on-demand crawl through corpusrecall, waits for the markdown to be
// stored, and returns it in the Firecrawl scrape response shape.
//
// The test is guarded by the e2e build tag and needs a working Docker daemon.
// Run it with `make e2e-firecrawlshim`. It is not part of the `make verify` gate.
package e2e
