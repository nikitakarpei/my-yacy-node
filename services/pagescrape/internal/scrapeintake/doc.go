// Package scrapeintake fetches each page a scrape request names, offers it to every corpus,
// and settles the request: a page the origin serves is offered, a page the origin defers
// comes back for a later fetch until the deferral window is spent, and anything else is
// reported as a scrape failure. Every request is settled once, so no page is fetched twice.
// It takes the next request only when its admission lets it.
package scrapeintake
