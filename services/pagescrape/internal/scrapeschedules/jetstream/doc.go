// Package jetstream hands a deferred scrape request back to the broker to redeliver at the
// time the origin asked for. The request waits in the stream, not in the service, so a page
// the origin holds back keeps no intake slot.
package jetstream
