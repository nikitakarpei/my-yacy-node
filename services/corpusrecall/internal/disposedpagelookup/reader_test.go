package disposedpagelookup_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/disposedpagelookup"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type stubEntry struct {
	revision uint64
}

func (e stubEntry) Bucket() string                  { return "" }
func (e stubEntry) Key() string                     { return "" }
func (e stubEntry) Value() []byte                   { return nil }
func (e stubEntry) Revision() uint64                { return e.revision }
func (e stubEntry) Created() time.Time              { return time.Time{} }
func (e stubEntry) Delta() uint64                   { return 0 }
func (e stubEntry) Operation() jetstream.KeyValueOp { return jetstream.KeyValuePut }

type fakeDisposedPages struct {
	entry jetstream.KeyValueEntry
	err   error
	key   string
}

func (f *fakeDisposedPages) Get(_ context.Context, key string) (jetstream.KeyValueEntry, error) {
	f.key = key
	return f.entry, f.err
}

func TestRevisionReturnsZeroWhenKeyMissing(t *testing.T) {
	const canonical = "https://example.com/"
	disposed := &fakeDisposedPages{err: jetstream.ErrKeyNotFound}
	reader := disposedpagelookup.NewReader(disposed)

	got, err := reader.Revision(context.Background(), canonical)
	if err != nil {
		t.Fatalf("revision: %v", err)
	}
	if got != 0 {
		t.Errorf("revision = %d, want 0", got)
	}
	if disposed.key != yacycrawlcontract.DisposedPageKey(canonical) {
		t.Errorf("looked up key %q", disposed.key)
	}
}

func TestRevisionReturnsStoredRevision(t *testing.T) {
	reader := disposedpagelookup.NewReader(&fakeDisposedPages{entry: stubEntry{revision: 7}})

	got, err := reader.Revision(context.Background(), "https://example.com/")
	if err != nil {
		t.Fatalf("revision: %v", err)
	}
	if got != 7 {
		t.Errorf("revision = %d, want 7", got)
	}
}

func TestRevisionPropagatesLookupError(t *testing.T) {
	reader := disposedpagelookup.NewReader(&fakeDisposedPages{err: errors.New("kv down")})

	if _, err := reader.Revision(context.Background(), "https://example.com/"); err == nil {
		t.Fatal("expected error from failing lookup")
	}
}
