package pagereplay_test

import (
	"net/http"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/pagereplay"
)

func TestCaptureTermsOfStateOnlyTheReplayCaptureTerms(t *testing.T) {
	servedResponseHeader := http.Header{}
	servedResponseHeader.Set("Memento-Datetime", "Mon, 24 Aug 2026 13:48:01 GMT")
	servedResponseHeader.Set("Link", `<https://example.com/>; rel="original"`)
	servedResponseHeader.Set("X-Archive-Orig-ETag", `"origin-tag"`)
	servedResponseHeader.Set("X-Archive-Orig-Last-Modified", "Tue, 18 Aug 2026 20:06:42 GMT")
	servedResponseHeader.Set("X-Archive-Orig-Set-Cookie", "session=secret")

	clientResponseHeader := http.Header{}
	pagereplay.CaptureTermsOf(servedResponseHeader).StateOn(clientResponseHeader)

	if got := clientResponseHeader.Get("Memento-Datetime"); got == "" {
		t.Fatal("memento-datetime is missing")
	}
	if got := clientResponseHeader.Get("Link"); got != `<https://example.com/>; rel="original"` {
		t.Fatalf("link = %q", got)
	}
	if got := clientResponseHeader.Get("X-Archive-Orig-ETag"); got != `"origin-tag"` {
		t.Fatalf("x-archive-orig-etag = %q", got)
	}
	if got := clientResponseHeader.Get("X-Archive-Orig-Last-Modified"); got == "" {
		t.Fatal("x-archive-orig-last-modified is missing")
	}
	if got := clientResponseHeader.Get("X-Archive-Orig-Set-Cookie"); got != "" {
		t.Fatalf("x-archive-orig-set-cookie = %q, want no value", got)
	}
}

func TestCaptureTermsOfStateNothingForAPageThatIsNoReplay(t *testing.T) {
	servedResponseHeader := http.Header{}
	servedResponseHeader.Set("ETag", `"v1"`)

	clientResponseHeader := http.Header{}
	pagereplay.CaptureTermsOf(servedResponseHeader).StateOn(clientResponseHeader)

	if len(clientResponseHeader) != 0 {
		t.Fatalf("client response header = %v, want no fields", clientResponseHeader)
	}
}

func TestCaptureTermsOfReplaceAFieldTheClientResponseAlreadyCarries(t *testing.T) {
	servedResponseHeader := http.Header{}
	servedResponseHeader.Set("Link", `<https://example.com/>; rel="original"`)

	clientResponseHeader := http.Header{}
	clientResponseHeader.Set("Link", `<https://elsewhere.example/>; rel="original"`)
	pagereplay.CaptureTermsOf(servedResponseHeader).StateOn(clientResponseHeader)

	if values := clientResponseHeader.Values("Link"); len(values) != 1 {
		t.Fatalf("link = %q, want one field", values)
	}
	if got := clientResponseHeader.Get("Link"); got != `<https://example.com/>; rel="original"` {
		t.Fatalf("link = %q", got)
	}
}
