package contentextraction_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentextraction"
)

type fakeExtractor struct {
	document contentextraction.ExtractedDocument
	err      error
	gotURL   string
	calls    int
}

func (f *fakeExtractor) Extract(
	_ context.Context,
	pageURL, _ string,
	_ []byte,
) (contentextraction.ExtractedDocument, error) {
	f.gotURL = pageURL
	f.calls++
	return f.document, f.err
}

func newExtraction(
	t *testing.T,
	extractors map[string]contentextraction.MediaExtractor,
) *contentextraction.DocumentExtractor {
	t.Helper()
	extractor, err := contentextraction.New(extractors)
	if err != nil {
		t.Fatalf("build extractor: %v", err)
	}
	return extractor
}

func TestTheExtractorRegisteredForTheMediaTypeReadsTheBody(t *testing.T) {
	extractor := &fakeExtractor{
		document: contentextraction.ExtractedDocument{Title: "page"},
	}
	documentExtractor := newExtraction(t,
		map[string]contentextraction.MediaExtractor{"text/html": extractor},
	)

	document, err := documentExtractor.DocumentFrom(
		t.Context(), "http://host/", "text/html", []byte("<html></html>"),
	)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if document.Title != "page" {
		t.Fatalf("want the extracted title, got %q", document.Title)
	}
	if extractor.gotURL != "http://host/" {
		t.Fatalf("want the page url, got %q", extractor.gotURL)
	}
}

func TestTheMediaTypeIsReadWithoutItsParameters(t *testing.T) {
	extractor := &fakeExtractor{
		document: contentextraction.ExtractedDocument{Title: "page"},
	}
	documentExtractor := newExtraction(t,
		map[string]contentextraction.MediaExtractor{"text/html": extractor},
	)

	if _, err := documentExtractor.DocumentFrom(
		t.Context(), "http://host/", "text/html; charset=utf-8", nil,
	); err != nil {
		t.Fatalf("extract: %v", err)
	}
	if extractor.calls != 1 {
		t.Fatalf("want the extractor called once, got %d calls", extractor.calls)
	}
}

func TestAMediaTypeNoExtractorReadsIsUnsupported(t *testing.T) {
	documentExtractor := newExtraction(t,
		map[string]contentextraction.MediaExtractor{"text/html": &fakeExtractor{}},
	)

	_, err := documentExtractor.DocumentFrom(
		t.Context(), "http://host/page.pdf", "application/pdf", nil,
	)
	if !errors.Is(err, contentextraction.ErrUnsupportedMediaType) {
		t.Fatalf("want ErrUnsupportedMediaType, got %v", err)
	}
}

func TestAFailingExtractorFailsTheExtraction(t *testing.T) {
	failure := errors.New("broken document")
	documentExtractor := newExtraction(t,
		map[string]contentextraction.MediaExtractor{
			"text/html": &fakeExtractor{err: failure},
		},
	)

	_, err := documentExtractor.DocumentFrom(t.Context(), "http://host/", "text/html", nil)
	if !errors.Is(err, failure) {
		t.Fatalf("want the extractor failure, got %v", err)
	}
}

func TestAnExtractorlessCatalogIsRejected(t *testing.T) {
	if _, err := contentextraction.New(nil); !errors.Is(
		err, contentextraction.ErrNoExtractableMediaType,
	) {
		t.Fatalf("want ErrNoExtractableMediaType, got %v", err)
	}
}
