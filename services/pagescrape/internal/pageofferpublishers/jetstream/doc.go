// Package jetstream offers each page the service reads, and each scrape it gives up on, to
// the corpora on the page offer stream. A corpus that is away when a page is offered still
// reads it later: the stream holds what no consumer has taken yet.
package jetstream
