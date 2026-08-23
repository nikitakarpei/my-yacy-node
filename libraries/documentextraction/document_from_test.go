package documentextraction_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

func TestTheExtractorRegisteredForTheMediaTypeReadsTheBody(t *testing.T) {
	document, err := documentextraction.DocumentFrom(
		t.Context(),
		[]byte("<html><head><title>page</title></head><body>text</body></html>"),
		"text/html",
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if document.Title != "page" {
		t.Fatalf("want the extracted title, got %q", document.Title)
	}
}

func TestTheMediaTypeIsReadWithoutItsParameters(t *testing.T) {
	document, err := documentextraction.DocumentFrom(
		t.Context(),
		[]byte("<html><head><title>page</title></head><body>text</body></html>"),
		"text/html; charset=utf-8",
		canonicalurltest.CanonicalURLOf(t, "http://host/"),
	)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if document.Title != "page" {
		t.Fatalf("want the extracted title, got %q", document.Title)
	}
}

func TestAMediaTypeNoExtractorReadsIsUnsupported(t *testing.T) {
	_, err := documentextraction.DocumentFrom(
		t.Context(), nil, "application/pdf",
		canonicalurltest.CanonicalURLOf(t, "http://host/page.pdf"),
	)
	if !errors.Is(err, documentextraction.ErrUnsupportedMediaType) {
		t.Fatalf("want ErrUnsupportedMediaType, got %v", err)
	}
}
