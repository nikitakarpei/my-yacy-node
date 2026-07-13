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

	got := outcome.result()
	if !got.seen || got.statusCode != 200 || got.contentType != "text/html" ||
		got.requestID != "1" {
		t.Fatalf("got %+v, want {200 text/html 1 true}", got)
	}
}

func TestRedirectChainSettlesOnFinalResponse(t *testing.T) {
	var outcome mainDocumentResponse
	outcome.observe(requestWillBeSent("1", network.ResourceTypeDocument))
	outcome.observe(responseReceived("1", 302, "text/html", "http://origin/start"))
	outcome.observe(responseReceived("1", 200, "text/html", "http://origin/final"))

	got := outcome.result()
	if !got.seen || got.statusCode != 200 {
		t.Fatalf("got %+v, want statusCode 200 seen true", got)
	}
}

func TestSubframeDocumentIgnored(t *testing.T) {
	var outcome mainDocumentResponse
	outcome.observe(requestWillBeSent("1", network.ResourceTypeDocument))
	outcome.observe(responseReceived("1", 200, "text/html", "http://origin/page"))
	outcome.observe(requestWillBeSent("2", network.ResourceTypeDocument))
	outcome.observe(responseReceived("2", 500, "text/html", "http://origin/frame"))

	got := outcome.result()
	if !got.seen || got.statusCode != 200 {
		t.Fatalf("got %+v, want statusCode 200 seen true", got)
	}
}

func TestNonDocumentRequestNotBound(t *testing.T) {
	var outcome mainDocumentResponse
	outcome.observe(requestWillBeSent("1", network.ResourceTypeXHR))
	outcome.observe(responseReceived("1", 200, "application/json", "http://origin/api"))

	if outcome.result().seen {
		t.Fatal("XHR request must not bind the main document")
	}
}

func TestNoDocumentResponse(t *testing.T) {
	var outcome mainDocumentResponse
	outcome.observe(requestWillBeSent("1", network.ResourceTypeDocument))

	if outcome.result().seen {
		t.Fatal("without a response, result must report not ok")
	}
}
