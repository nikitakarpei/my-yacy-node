// Package jetstreamrecord reads a record from a key-value bucket and writes a
// revision of it back only when no other writer changed the record first. It
// waits out the writers that win the key, and gives up after a bounded number
// of attempts.
package jetstreamrecord
