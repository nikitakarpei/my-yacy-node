package cdprender

import (
	"testing"

	"github.com/chromedp/cdproto/network"
)

func requestWillBeSent(
	id string,
	resourceType network.ResourceType,
) *network.EventRequestWillBeSent {
	return &network.EventRequestWillBeSent{
		RequestID: network.RequestID(id),
		Type:      resourceType,
	}
}

func responseReceived(
	id string,
	status int64,
	mimeType, url string,
) *network.EventResponseReceived {
	return &network.EventResponseReceived{
		RequestID: network.RequestID(id),
		Response:  &network.Response{Status: status, MimeType: mimeType, URL: url},
	}
}

func TestPlainDocumentResponse(t *testing.T) {
	var outcome mainDocumentResponse
	outcome.observe(requestWillBeSent("1", network.ResourceTypeDocument))
	outcome.observe(responseReceived("1", 200, "text/html", "http://origin/page"))

	status, contentType, ok := outcome.result()
	if !ok || status != 200 || contentType != "text/html" {
		t.Fatalf("got (%d, %q, %v), want (200, text/html, true)", status, contentType, ok)
	}
}

func TestRedirectChainSettlesOnFinalResponse(t *testing.T) {
	var outcome mainDocumentResponse
	outcome.observe(requestWillBeSent("1", network.ResourceTypeDocument))
	outcome.observe(responseReceived("1", 302, "text/html", "http://origin/start"))
	outcome.observe(responseReceived("1", 200, "text/html", "http://origin/final"))

	status, _, ok := outcome.result()
	if !ok || status != 200 {
		t.Fatalf("got (%d, %v), want (200, true)", status, ok)
	}
}

func TestSubframeDocumentIgnored(t *testing.T) {
	var outcome mainDocumentResponse
	outcome.observe(requestWillBeSent("1", network.ResourceTypeDocument))
	outcome.observe(responseReceived("1", 200, "text/html", "http://origin/page"))
	outcome.observe(requestWillBeSent("2", network.ResourceTypeDocument))
	outcome.observe(responseReceived("2", 500, "text/html", "http://origin/frame"))

	status, _, ok := outcome.result()
	if !ok || status != 200 {
		t.Fatalf("got (%d, %v), want (200, true)", status, ok)
	}
}

func TestNonDocumentRequestNotBound(t *testing.T) {
	var outcome mainDocumentResponse
	outcome.observe(requestWillBeSent("1", network.ResourceTypeXHR))
	outcome.observe(responseReceived("1", 200, "application/json", "http://origin/api"))

	if _, _, ok := outcome.result(); ok {
		t.Fatal("XHR request must not bind the main document")
	}
}

func TestNoDocumentResponse(t *testing.T) {
	var outcome mainDocumentResponse
	outcome.observe(requestWillBeSent("1", network.ResourceTypeDocument))

	if _, _, ok := outcome.result(); ok {
		t.Fatal("without a response, result must report not ok")
	}
}
