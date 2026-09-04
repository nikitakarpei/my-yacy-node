// Package jetstream holds what a crawl order has already taken for one page visit
// — a page of the URL's host, the deferrals of that URL and its attempts — and
// admits another only while the order's limit leaves room for it. A redelivered
// page visit takes the page of its host once.
package jetstream
