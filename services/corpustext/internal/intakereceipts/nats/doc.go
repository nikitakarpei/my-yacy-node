// Package nats sends back what this corpus did with each page the scrape service offered:
// it kept the page, or it rejected it. Every receipt goes on the subject of that one page,
// where the scrape service picks it up. Nothing keeps a receipt.
package nats
