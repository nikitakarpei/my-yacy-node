// Package redis keeps the frontier on a Redis-protocol server: every page visit a crawl
// order still owes, each due at a time. A worker takes a due visit for a lease, and the
// visit comes back to the frontier when the lease ends before the worker settles it.
// A visit already pending is not added twice.
package redis
