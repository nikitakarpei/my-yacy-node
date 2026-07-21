package redirectlookup_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/redirectlookup"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type stubEntry struct {
	value []byte
}

func (e stubEntry) Bucket() string                  { return "" }
func (e stubEntry) Key() string                     { return "" }
func (e stubEntry) Value() []byte                   { return e.value }
func (e stubEntry) Revision() uint64                { return 0 }
func (e stubEntry) Created() time.Time              { return time.Time{} }
func (e stubEntry) Delta() uint64                   { return 0 }
func (e stubEntry) Operation() jetstream.KeyValueOp { return jetstream.KeyValuePut }

type fakeRedirects struct {
	entry jetstream.KeyValueEntry
	err   error
	key   string
}

func (f *fakeRedirects) Get(_ context.Context, key string) (jetstream.KeyValueEntry, error) {
	f.key = key
	return f.entry, f.err
}

func TestResolveFallsBackToCanonicalWhenKeyMissing(t *testing.T) {
	const canonical = "https://example.com/"
	redirects := &fakeRedirects{err: jetstream.ErrKeyNotFound}
	reader := redirectlookup.NewReader(redirects)

	got, err := reader.Resolve(context.Background(), canonical)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != canonical {
		t.Errorf("resolved = %q, want %q", got, canonical)
	}
	if redirects.key != yacycrawlcontract.RedirectResolutionKey(canonical) {
		t.Errorf("looked up key %q", redirects.key)
	}
}

func TestResolveReturnsStoredTarget(t *testing.T) {
	const target = "https://example.com/final"
	reader := redirectlookup.NewReader(&fakeRedirects{entry: stubEntry{value: []byte(target)}})

	got, err := reader.Resolve(context.Background(), "https://example.com/")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != target {
		t.Errorf("resolved = %q, want %q", got, target)
	}
}

func TestResolvePropagatesLookupError(t *testing.T) {
	reader := redirectlookup.NewReader(&fakeRedirects{err: errors.New("kv down")})

	if _, err := reader.Resolve(context.Background(), "https://example.com/"); err == nil {
		t.Fatal("expected error from failing lookup")
	}
}
