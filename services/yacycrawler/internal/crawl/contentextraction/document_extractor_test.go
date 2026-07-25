package contentextraction_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentextraction"
)

type fakeExtractor struct {
	content contentextraction.ExtractedContent
	err     error
	gotURL  string
	calls   int
}

func (f *fakeExtractor) Extract(
	_ context.Context,
	pageURL, _ string,
	_ []byte,
) (contentextraction.ExtractedContent, error) {
	f.gotURL = pageURL
	f.calls++
	return f.content, f.err
}

type fakeContainer struct {
	members []contentextraction.ContainerMember
	err     error
}

func (f *fakeContainer) Expand(
	_ context.Context,
	_, _ string,
	_ []byte,
) ([]contentextraction.ContainerMember, error) {
	return f.members, f.err
}

func newExtraction(
	t *testing.T,
	extractors map[string]contentextraction.MediaExtractor,
	containers map[string]contentextraction.ContainerExpander,
	maxDepth, maxDocuments int,
) *contentextraction.DocumentExtractor {
	t.Helper()
	extractor, err := contentextraction.New(extractors, containers, maxDepth, maxDocuments)
	if err != nil {
		t.Fatalf("build extractor: %v", err)
	}
	return extractor
}

func TestExtractDispatchesToRegisteredExtractor(t *testing.T) {
	extractor := &fakeExtractor{content: contentextraction.ExtractedContent{Title: "page"}}
	documentExtractor := newExtraction(t,
		map[string]contentextraction.MediaExtractor{"text/html": extractor},
		nil, 4, 16,
	)

	documents, err := documentExtractor.ExtractDocuments(
		t.Context(),
		"http://host/p",
		"text/html; charset=utf-8",
		[]byte("x"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(documents) != 1 || documents[0].Title != "page" {
		t.Fatalf("unexpected documents: %+v", documents)
	}
	if extractor.gotURL != "http://host/p" {
		t.Fatalf("extractor got url %q", extractor.gotURL)
	}
}

func TestExtractUnsupportedMediaType(t *testing.T) {
	documentExtractor := newExtraction(t,
		map[string]contentextraction.MediaExtractor{"text/html": &fakeExtractor{}},
		nil, 4, 16,
	)

	_, err := documentExtractor.ExtractDocuments(
		t.Context(),
		"http://host/f",
		"application/pdf",
		[]byte("x"),
	)
	if !errors.Is(err, contentextraction.ErrUnsupportedMediaType) {
		t.Fatalf("want ErrUnsupportedMediaType, got %v", err)
	}
}

func TestExtractExpandsContainerAndStampsMemberURL(t *testing.T) {
	html := &fakeExtractor{content: contentextraction.ExtractedContent{Title: "member"}}
	container := &fakeContainer{members: []contentextraction.ContainerMember{
		{URL: "http://host/a.zip!/one.html", ContentType: "text/html", Body: []byte("1")},
		{URL: "http://host/a.zip!/skip.bin", ContentType: "application/octet-stream"},
	}}
	documentExtractor := newExtraction(t,
		map[string]contentextraction.MediaExtractor{"text/html": html},
		map[string]contentextraction.ContainerExpander{"application/zip": container},
		4, 16,
	)

	documents, err := documentExtractor.ExtractDocuments(
		t.Context(),
		"http://host/a.zip",
		"application/zip",
		[]byte("x"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(documents) != 1 {
		t.Fatalf("want 1 document (unsupported member skipped), got %d", len(documents))
	}
	if documents[0].URL != "http://host/a.zip!/one.html" {
		t.Fatalf("member url not stamped: %q", documents[0].URL)
	}
}

func TestExtractNestedContainerExpands(t *testing.T) {
	html := &fakeExtractor{content: contentextraction.ExtractedContent{Title: "deep"}}
	tar := &fakeContainer{members: []contentextraction.ContainerMember{
		{URL: "u!/inner.tar!/p.html", ContentType: "text/html"},
	}}
	zip := &fakeContainer{members: []contentextraction.ContainerMember{
		{URL: "u!/inner.tar", ContentType: "application/x-tar"},
	}}
	documentExtractor := newExtraction(t,
		map[string]contentextraction.MediaExtractor{"text/html": html},
		map[string]contentextraction.ContainerExpander{
			"application/zip":   zip,
			"application/x-tar": tar,
		},
		4, 16,
	)

	documents, err := documentExtractor.ExtractDocuments(
		t.Context(),
		"u",
		"application/zip",
		[]byte("x"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(documents) != 1 || documents[0].Title != "deep" {
		t.Fatalf("nested expansion unexpected: %+v", documents)
	}
}

func TestExtractNestingDepthOverflow(t *testing.T) {
	selfContainer := &fakeContainer{members: []contentextraction.ContainerMember{
		{URL: "u!/again.zip", ContentType: "application/zip"},
	}}
	documentExtractor := newExtraction(t, nil,
		map[string]contentextraction.ContainerExpander{"application/zip": selfContainer},
		2, 16,
	)

	_, err := documentExtractor.ExtractDocuments(t.Context(), "u", "application/zip", []byte("x"))
	if !errors.Is(err, contentextraction.ErrNestingTooDeep) {
		t.Fatalf("want ErrNestingTooDeep, got %v", err)
	}
}

func TestExtractDocumentBudgetOverflow(t *testing.T) {
	html := &fakeExtractor{content: contentextraction.ExtractedContent{Title: "m"}}
	container := &fakeContainer{members: []contentextraction.ContainerMember{
		{URL: "u!/1.html", ContentType: "text/html"},
		{URL: "u!/2.html", ContentType: "text/html"},
		{URL: "u!/3.html", ContentType: "text/html"},
	}}
	documentExtractor := newExtraction(t,
		map[string]contentextraction.MediaExtractor{"text/html": html},
		map[string]contentextraction.ContainerExpander{"application/zip": container},
		4, 2,
	)

	_, err := documentExtractor.ExtractDocuments(t.Context(), "u", "application/zip", []byte("x"))
	if !errors.Is(err, contentextraction.ErrDocumentBudgetExhausted) {
		t.Fatalf("want ErrDocumentBudgetExhausted, got %v", err)
	}
}

func TestExtractContainerExpandError(t *testing.T) {
	container := &fakeContainer{err: errors.New("corrupt")}
	documentExtractor := newExtraction(t, nil,
		map[string]contentextraction.ContainerExpander{"application/zip": container},
		4, 16,
	)

	_, err := documentExtractor.ExtractDocuments(t.Context(), "u", "application/zip", []byte("x"))
	if err == nil {
		t.Fatal("want error from expand")
	}
}

func TestNewRejectsEmptyMediaTypes(t *testing.T) {
	_, err := contentextraction.New(nil, nil, 4, 16)
	if !errors.Is(err, contentextraction.ErrNoExtractableMediaType) {
		t.Fatalf("want ErrNoExtractableMediaType, got %v", err)
	}
}

func TestExtractFallsBackWhenContentTypeUnparsed(t *testing.T) {
	extractor := &fakeExtractor{content: contentextraction.ExtractedContent{Title: "page"}}
	documentExtractor := newExtraction(t,
		map[string]contentextraction.MediaExtractor{"text/html": extractor},
		nil, 4, 16,
	)

	documents, err := documentExtractor.ExtractDocuments(
		t.Context(),
		"u",
		"Text/HTML ;;",
		[]byte("x"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(documents) != 1 {
		t.Fatalf("want the leading segment to route, got %+v", documents)
	}
}

func TestExtractStopsAtDocumentBudgetAcrossNestedContainers(t *testing.T) {
	html := &fakeExtractor{content: contentextraction.ExtractedContent{Title: "m"}}
	tar := &fakeContainer{members: []contentextraction.ContainerMember{
		{URL: "u!/inner.tar!/1.html", ContentType: "text/html"},
		{URL: "u!/inner.tar!/2.html", ContentType: "text/html"},
	}}
	zip := &fakeContainer{members: []contentextraction.ContainerMember{
		{URL: "u!/first.tar", ContentType: "application/x-tar"},
		{URL: "u!/second.tar", ContentType: "application/x-tar"},
	}}
	documentExtractor := newExtraction(t,
		map[string]contentextraction.MediaExtractor{"text/html": html},
		map[string]contentextraction.ContainerExpander{
			"application/zip":   zip,
			"application/x-tar": tar,
		},
		4, 3,
	)

	_, err := documentExtractor.ExtractDocuments(t.Context(), "u", "application/zip", []byte("x"))
	if !errors.Is(err, contentextraction.ErrDocumentBudgetExhausted) {
		t.Fatalf("want ErrDocumentBudgetExhausted, got %v", err)
	}
	if html.calls != 3 {
		t.Fatalf("extraction should stop at the document budget, ran %d times", html.calls)
	}
}
