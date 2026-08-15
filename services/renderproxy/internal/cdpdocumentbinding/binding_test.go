package cdpdocumentbinding_test

import (
	"net/http"
	"testing"

	"github.com/chromedp/cdproto/network"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/cdpdocumentbinding"
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

func TestBindingCarriesTheDocumentStatusAndContentType(t *testing.T) {
	var outcome cdpdocumentbinding.Binding
	outcome.Observe(requestWillBeSent("1", network.ResourceTypeDocument))
	outcome.Observe(responseReceived("1", http.StatusOK, "text/html", "http://origin/page"))

	got := outcome.BoundDocument()
	if !got.Seen || got.StatusCode != http.StatusOK || got.ContentType != "text/html" ||
		got.RequestID != "1" {
		t.Fatalf("got %+v, want {200 text/html 1 true}", got)
	}
}

func TestBindingKeepsTheLatestStatusForTheBoundRequest(t *testing.T) {
	var outcome cdpdocumentbinding.Binding
	outcome.Observe(requestWillBeSent("1", network.ResourceTypeDocument))
	outcome.Observe(responseReceived("1", http.StatusFound, "text/html", "http://origin/start"))
	outcome.Observe(responseReceived("1", http.StatusOK, "text/html", "http://origin/final"))

	got := outcome.BoundDocument()
	if !got.Seen || got.StatusCode != http.StatusOK {
		t.Fatalf("got %+v, want statusCode 200 seen true", got)
	}
}

func TestBindingIgnoresLaterDocumentRequests(t *testing.T) {
	var outcome cdpdocumentbinding.Binding
	outcome.Observe(requestWillBeSent("1", network.ResourceTypeDocument))
	outcome.Observe(responseReceived("1", http.StatusOK, "text/html", "http://origin/page"))
	outcome.Observe(requestWillBeSent("2", network.ResourceTypeDocument))
	outcome.Observe(
		responseReceived("2", http.StatusInternalServerError, "text/html", "http://origin/frame"),
	)

	got := outcome.BoundDocument()
	if !got.Seen || got.StatusCode != http.StatusOK {
		t.Fatalf("got %+v, want statusCode 200 seen true", got)
	}
}

func TestBindingIgnoresNonDocumentRequests(t *testing.T) {
	var outcome cdpdocumentbinding.Binding
	outcome.Observe(requestWillBeSent("1", network.ResourceTypeXHR))
	outcome.Observe(responseReceived("1", http.StatusOK, "application/json", "http://origin/api"))

	if outcome.BoundDocument().Seen {
		t.Fatal("XHR request must not bind the main document")
	}
}

func TestBindingReportsNoDocumentWithoutAResponse(t *testing.T) {
	var outcome cdpdocumentbinding.Binding
	outcome.Observe(requestWillBeSent("1", network.ResourceTypeDocument))

	if outcome.BoundDocument().Seen {
		t.Fatal("without a response, result must report not ok")
	}
}
