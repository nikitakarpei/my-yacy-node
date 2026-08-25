// Package pagereplay carries what a replay states about the capture it serves across the
// proxy: the moment of capture, the url the bytes were captured from, and the validators
// the origin gave at capture time.
package pagereplay

import "net/http"

type CaptureTerms http.Header

var captureTermNames = []string{
	"Memento-Datetime",
	"Link",
	"X-Archive-Orig-ETag",
	"X-Archive-Orig-Last-Modified",
}

func CaptureTermsOf(servedResponseHeader http.Header) CaptureTerms {
	fields := http.Header{}
	for _, name := range captureTermNames {
		for _, value := range servedResponseHeader.Values(name) {
			fields.Add(name, value)
		}
	}
	return CaptureTerms(fields)
}

func (t CaptureTerms) StateOn(clientResponseHeader http.Header) {
	for name, values := range t {
		clientResponseHeader.Del(name)
		for _, value := range values {
			clientResponseHeader.Add(name, value)
		}
	}
}
