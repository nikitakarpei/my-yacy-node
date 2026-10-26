// Package e2e runs the end-to-end local index search test against containers.
//
// It starts two origin pages behind an egress proxy and corpustext, publishes one scrape
// request per origin, and starts the real searxng/searxng image with the local index
// settings of the build-your-own-search-stack guide. It then drives searches from the
// host and checks that the results carry the indexed title, URL, and matched content,
// in the language the search names.
//
// The test is guarded by the e2e build tag and needs a working Docker
// daemon. Run it with `make e2e-searxnglocalindex`. It is not
// part of the `make verify` quality gate.
package e2e
