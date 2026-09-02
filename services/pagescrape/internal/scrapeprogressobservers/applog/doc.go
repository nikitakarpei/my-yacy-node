// Package applog writes a log line for each fact the service learns about a scrape it takes
// on, so an operator can read the whole life of one request in the service log. It is the
// only place that decides how a fact reads and at which level it is written.
package applog
