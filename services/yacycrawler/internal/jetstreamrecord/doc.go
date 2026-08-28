// Package jetstreamrecord opens the key-value buckets a crawl keeps its records
// in, addresses a record by a key that hides what it was made from, and writes a
// revision of a record back only when no other writer changed it first. It waits
// out the writers that win a key, and gives up after a bounded number of
// attempts.
package jetstreamrecord
