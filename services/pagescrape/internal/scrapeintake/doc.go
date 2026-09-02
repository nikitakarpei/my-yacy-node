// Package scrapeintake reads each page a scrape request names, offers it to every corpus, and
// settles the request: a page that is read is offered, a page the origin defers is scheduled
// for a later read until the deferral window is spent, and anything else is reported as a
// scrape failure. Every request is settled once, so no request is read twice.
package scrapeintake
