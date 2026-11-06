// Package jetstream creates the stream of the RWIs of pages, which this service owns. The stream
// keeps an RWI only while a node consumer has still to take it, and refuses new ones while full.
package jetstream
