package pagehtml_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
)

const page = `<html><head><title>t</title></head><body><a href="/first">first</a><a href="/second">second</a></body></html>`

func TestElementTreeFromRejectsABodyThatIsNotHTML(t *testing.T) {
	parser := pagehtml.NewHTMLParser(&recordingMediaTypeObserver{})

	if _, err := parser.ElementTreeFrom(
		t.Context(), "application/pdf", []byte(page),
	); !errors.Is(err, pagehtml.ErrNotHTML) {
		t.Fatalf("want ErrNotHTML, got %v", err)
	}
}

func TestElementTreeFromAcceptsTheHTMLMediaTypes(t *testing.T) {
	parser := pagehtml.NewHTMLParser(&recordingMediaTypeObserver{})

	for _, contentType := range []string{
		"text/html",
		"text/html; charset=utf-8",
		"application/xhtml+xml",
		"text/html;",
	} {
		if _, err := parser.ElementTreeFrom(
			t.Context(), contentType, []byte(page),
		); err != nil {
			t.Errorf("ElementTreeFrom %q: %v", contentType, err)
		}
	}
}

func TestElementTreeFromReportsAnUnreadableCharset(t *testing.T) {
	parser := pagehtml.NewHTMLParser(&recordingMediaTypeObserver{})

	_, err := parser.ElementTreeFrom(
		t.Context(), "text/html", nil,
	)

	if !errors.Is(err, pagehtml.ErrCharsetUnreadable) {
		t.Fatalf("want ErrCharsetUnreadable, got %v", err)
	}
}

func TestAContentTypeThatCannotBeParsedIsObserved(t *testing.T) {
	observer := &recordingMediaTypeObserver{}
	parser := pagehtml.NewHTMLParser(observer)

	if _, err := parser.ElementTreeFrom(
		t.Context(), "text/html; charset", []byte(page),
	); err != nil {
		t.Fatalf("ElementTreeFrom: %v", err)
	}

	if len(observer.unparsedContentTypes) != 1 ||
		observer.unparsedContentTypes[0] != "text/html; charset" {
		t.Fatalf("unparsed content types = %v", observer.unparsedContentTypes)
	}
}

func TestEveryObserverHearsAContentTypeThatCannotBeParsed(t *testing.T) {
	first, second := &recordingMediaTypeObserver{}, &recordingMediaTypeObserver{}
	parser := pagehtml.NewHTMLParser(pagehtml.MediaTypeObservers{first, second})

	if _, err := parser.ElementTreeFrom(
		t.Context(), "text/html; charset", []byte(page),
	); err != nil {
		t.Fatalf("ElementTreeFrom: %v", err)
	}

	if len(first.unparsedContentTypes) != 1 || len(second.unparsedContentTypes) != 1 {
		t.Fatalf(
			"unparsed content types = %v and %v, want one each",
			first.unparsedContentTypes, second.unparsedContentTypes,
		)
	}
}

type recordingMediaTypeObserver struct {
	unparsedContentTypes []string
}

func (o *recordingMediaTypeObserver) MediaTypeUnparsed(
	_ context.Context,
	contentType string,
	_ error,
) {
	o.unparsedContentTypes = append(o.unparsedContentTypes, contentType)
}
