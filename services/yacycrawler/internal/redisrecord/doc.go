// Package redisrecord addresses a crawl record by a key that names its kind and hides
// what it was made from, keeps the record for the retention of its kind, and writes a
// revision of a record back only when no other writer changed it first. It waits out
// the writers that win a key, and gives up after a bounded number of attempts.
package redisrecord
