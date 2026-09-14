// Package prometheus counts what became of every scrape the service took on, so an operator
// can tell an origin that serves nothing from a broker that takes nothing and leaves the
// request for a later retry, and can see which reason gives pages up.
package prometheus
