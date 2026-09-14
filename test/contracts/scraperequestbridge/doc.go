// Package scraperequestbridge holds the test that keeps a crawled page readable as a
// scrape request. The bridge consumer moves the body unchanged, so the two payloads must
// stay byte-compatible and no compiler checks that.
package scraperequestbridge
