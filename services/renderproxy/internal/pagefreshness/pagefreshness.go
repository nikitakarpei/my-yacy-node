// Package pagefreshness carries the cache freshness of a page across the proxy: the
// conditions a client states about the copy it already holds, and the terms an origin
// gives for reusing the copy it serves.
package pagefreshness

import "net/http"

type Conditions http.Header

type ReuseTerms http.Header

var conditionNames = []string{
	"If-None-Match",
	"If-Modified-Since",
}

var reuseTermNames = []string{
	"ETag",
	"Last-Modified",
	"Cache-Control",
	"Expires",
	"Age",
	"Vary",
}

func ConditionsOf(clientRequestHeader http.Header) Conditions {
	return Conditions(fieldsNamed(clientRequestHeader, conditionNames))
}

func ReuseTermsOf(servedResponseHeader http.Header) ReuseTerms {
	return ReuseTerms(fieldsNamed(servedResponseHeader, reuseTermNames))
}

func (c Conditions) StateOn(originRequestHeader http.Header) {
	stateFields(http.Header(c), originRequestHeader)
}

func (t ReuseTerms) StateOn(clientResponseHeader http.Header) {
	stateFields(http.Header(t), clientResponseHeader)
}

func fieldsNamed(header http.Header, names []string) http.Header {
	fields := http.Header{}
	for _, name := range names {
		for _, value := range header.Values(name) {
			fields.Add(name, value)
		}
	}
	return fields
}

func stateFields(fields http.Header, header http.Header) {
	for name, values := range fields {
		header.Del(name)
		for _, value := range values {
			header.Add(name, value)
		}
	}
}
