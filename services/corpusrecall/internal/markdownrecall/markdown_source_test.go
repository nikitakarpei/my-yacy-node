package markdownrecall_test

import (
	"context"
	"testing"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/markdownrecall"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
)

type fakeObjects struct {
	markdown []byte
	err      error
	name     string
}

func (f *fakeObjects) GetBytes(
	_ context.Context,
	name string,
	_ ...jetstream.GetObjectOpt,
) ([]byte, error) {
	f.name = name
	return f.markdown, f.err
}

func TestFetchReturnsMarkdownPage(t *testing.T) {
	const target = "https://example.com/"
	objects := &fakeObjects{markdown: []byte("# Hi")}
	source := markdownrecall.NewSource(objects, 1024)

	representation, found, err := source.Fetch(context.Background(), target)
	if err != nil || !found {
		t.Fatalf("fetch found=%v err=%v", found, err)
	}
	page, ok := representation.(markdownrecall.MarkdownPage)
	if !ok {
		t.Fatalf("representation type = %T", representation)
	}
	if page.CanonicalURL != target || page.Markdown != "# Hi" {
		t.Errorf("page = %+v", page)
	}
	if objects.name != pagemarkdownstore.ObjectName(target) {
		t.Errorf("object name = %q", objects.name)
	}
}

func TestFetchReportsMissingWhenObjectAbsent(t *testing.T) {
	source := markdownrecall.NewSource(&fakeObjects{err: jetstream.ErrObjectNotFound}, 1024)

	representation, found, err := source.Fetch(context.Background(), "https://example.com/")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if found || representation != nil {
		t.Errorf("found=%v representation=%v, want absent", found, representation)
	}
}

func TestFetchFailsWhenMarkdownExceedsLimit(t *testing.T) {
	source := markdownrecall.NewSource(&fakeObjects{markdown: []byte("toolong")}, 3)

	if _, _, err := source.Fetch(context.Background(), "https://example.com/"); err == nil {
		t.Fatal("expected error for oversize markdown")
	}
}
