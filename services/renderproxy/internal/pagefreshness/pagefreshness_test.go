package pagefreshness_test

import (
	"net/http"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/renderproxy/internal/pagefreshness"
)

func TestConditionsOfStatesOnlyTheClientConditions(t *testing.T) {
	clientRequestHeader := http.Header{}
	clientRequestHeader.Set("If-None-Match", `"v1"`)
	clientRequestHeader.Set("If-Modified-Since", "Mon, 02 Jan 2006 15:04:05 GMT")
	clientRequestHeader.Set("Cookie", "session=secret")

	originRequestHeader := http.Header{}
	pagefreshness.ConditionsOf(clientRequestHeader).StateOn(originRequestHeader)

	if got := originRequestHeader.Get("If-None-Match"); got != `"v1"` {
		t.Fatalf("if-none-match = %q", got)
	}
	if got := originRequestHeader.Get("If-Modified-Since"); got == "" {
		t.Fatal("if-modified-since is missing")
	}
	if got := originRequestHeader.Get("Cookie"); got != "" {
		t.Fatalf("cookie = %q, want no value", got)
	}
}

func TestReuseTermsOfStateOnlyTheOriginReuseTerms(t *testing.T) {
	originResponseHeader := http.Header{}
	originResponseHeader.Set("ETag", `"v1"`)
	originResponseHeader.Set("Cache-Control", "max-age=60")
	originResponseHeader.Set("Vary", "Accept-Encoding")
	originResponseHeader.Set("Set-Cookie", "session=secret")

	clientResponseHeader := http.Header{}
	pagefreshness.ReuseTermsOf(originResponseHeader).StateOn(clientResponseHeader)

	if got := clientResponseHeader.Get("ETag"); got != `"v1"` {
		t.Fatalf("etag = %q", got)
	}
	if got := clientResponseHeader.Get("Cache-Control"); got != "max-age=60" {
		t.Fatalf("cache-control = %q", got)
	}
	if got := clientResponseHeader.Get("Vary"); got != "Accept-Encoding" {
		t.Fatalf("vary = %q", got)
	}
	if got := clientResponseHeader.Get("Set-Cookie"); got != "" {
		t.Fatalf("set-cookie = %q, want no value", got)
	}
}

func TestReuseTermsOfReplaceAFieldTheClientResponseAlreadyCarries(t *testing.T) {
	originResponseHeader := http.Header{}
	originResponseHeader.Set("Cache-Control", "max-age=60")

	clientResponseHeader := http.Header{}
	clientResponseHeader.Set("Cache-Control", "no-store")
	pagefreshness.ReuseTermsOf(originResponseHeader).StateOn(clientResponseHeader)

	if values := clientResponseHeader.Values("Cache-Control"); len(values) != 1 {
		t.Fatalf("cache-control = %q, want one field", values)
	}
	if got := clientResponseHeader.Get("Cache-Control"); got != "max-age=60" {
		t.Fatalf("cache-control = %q", got)
	}
}
