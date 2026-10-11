// Package jetstream creates the two streams of a scrape, which this service owns: the one it
// takes requests from, which keeps a request, deferred or not, only until the service finishes
// it, and the one it offers pages on, which keeps a page only while a corpus has still to take
// it. Both refuse new messages while full.
package jetstream
