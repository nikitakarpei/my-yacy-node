// Package jetstream creates the two streams of a scrape, which this service owns: the one it
// takes requests from, which also holds the requests an origin deferred until the broker
// redelivers them, and the one it offers pages on, which keeps a page only while a corpus has
// still to take it.
package jetstream
