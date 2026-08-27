// Package scraperequestcontract names the stream, the subject, and the payload that ask
// every interested corpus to scrape a page.
//
// A request names the url that identifies the page and the url the bytes are read from. A
// request that names no url to read from is read from the url that identifies the page.
package scraperequestcontract

const (
	ScrapeRequestsStreamName = "SCRAPE_REQUESTS"
	ScrapeRequestSubject     = "scrape.request"
)
