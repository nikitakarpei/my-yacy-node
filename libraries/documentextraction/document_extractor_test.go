package documentextraction_test

import (
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
)

func TestTheExtractorRegisteredForTheMediaTypeReadsTheBody(t *testing.T) {
	document, err := documentextraction.New().DocumentFrom(
		t.Context(),
		"http://host/",
		"text/html",
		[]byte("<html><head><title>page</title></head><body>text</body></html>"),
	)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if document.Title != "page" {
		t.Fatalf("want the extracted title, got %q", document.Title)
	}
}

func TestTheMediaTypeIsReadWithoutItsParameters(t *testing.T) {
	document, err := documentextraction.New().DocumentFrom(
		t.Context(),
		"http://host/",
		"text/html; charset=utf-8",
		[]byte("<html><head><title>page</title></head><body>text</body></html>"),
	)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if document.Title != "page" {
		t.Fatalf("want the extracted title, got %q", document.Title)
	}
}

func TestAMediaTypeNoExtractorReadsIsUnsupported(t *testing.T) {
	_, err := documentextraction.New().DocumentFrom(
		t.Context(), "http://host/page.pdf", "application/pdf", nil,
	)
	if !errors.Is(err, documentextraction.ErrUnsupportedMediaType) {
		t.Fatalf("want ErrUnsupportedMediaType, got %v", err)
	}
}
