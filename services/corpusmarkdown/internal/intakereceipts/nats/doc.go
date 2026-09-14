// Package nats sends back what this corpus did with each page the scrape service offered:
// it kept the page, or it rejected it. Every receipt goes on the subject of that one page,
// where the scrape service picks it up. Nothing keeps a receipt, and the caller waiting on
// the page is told nothing when one does not arrive; how each receipt fared goes to the
// observer instead.
package nats
