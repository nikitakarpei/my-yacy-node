// Package scrapeintake fetches each page a scrape request names, offers it to every corpus,
// and settles the request: a page the origin serves is offered, a page the origin defers is
// scheduled for a later fetch until the deferral window is spent, and anything else is
// reported as a scrape failure. Every request is settled once, so no page is fetched twice.
package scrapeintake
