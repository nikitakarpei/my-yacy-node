package yacyseedlist_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/yacyseedlist"
)

const responseLimit = 1 << 20

type recordedRead struct {
	seeds       int
	unreachable int
	unreadable  int
}

func (r *recordedRead) SeedlistRead(_ context.Context, _ string, seeds int) { r.seeds += seeds }
func (r *recordedRead) SeedlistUnreachable(context.Context, string, error)  { r.unreachable++ }
func (r *recordedRead) SeedlistUnreadable(context.Context, string, error)   { r.unreadable++ }

func seedLine(hash, name, host string) string {
	return "Hash=" + hash + ",Name=" + name + ",PeerType=senior,IP=" + host + ",Port=8090"
}

func seedlistServing(t *testing.T, body string, status int) string {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(
		func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(status)
			_, _ = writer.Write([]byte(body))
		},
	))
	t.Cleanup(server.Close)

	return server.URL
}

func TestEverySeedlistAddsItsSeeds(t *testing.T) {
	t.Parallel()

	observer := &recordedRead{}
	seedlists := yacyseedlist.New(
		http.DefaultClient,
		[]string{
			seedlistServing(t, seedLine("aaaaaaaaaaaa", "first", "10.0.0.1"), http.StatusOK),
			seedlistServing(t, seedLine("bbbbbbbbbbbb", "second", "10.0.0.2"), http.StatusOK),
		},
		responseLimit,
		observer,
	)

	seeds := seedlists.Fetch(t.Context())

	if len(seeds) != 2 {
		t.Fatalf("Fetch = %+v, want one seed from each seedlist", seeds)
	}
	if observer.seeds != 2 {
		t.Fatalf("SeedlistRead reported %d seeds, want 2", observer.seeds)
	}
}

func TestASeedlistThatRefusesToAnswerAddsNoSeed(t *testing.T) {
	t.Parallel()

	observer := &recordedRead{}
	seedlists := yacyseedlist.New(
		http.DefaultClient,
		[]string{seedlistServing(t, "", http.StatusNotFound)},
		responseLimit,
		observer,
	)

	if seeds := seedlists.Fetch(t.Context()); len(seeds) != 0 {
		t.Fatalf("Fetch = %+v, want no seeds", seeds)
	}
	if observer.unreadable != 1 {
		t.Fatalf("SeedlistUnreadable reported %d times, want once", observer.unreadable)
	}
}

func TestASeedlistThatCannotBeReachedAddsNoSeed(t *testing.T) {
	t.Parallel()

	observer := &recordedRead{}
	seedlists := yacyseedlist.New(
		http.DefaultClient,
		[]string{"http://127.0.0.1:1/yacy/seedlist.html"},
		responseLimit,
		observer,
	)

	if seeds := seedlists.Fetch(t.Context()); len(seeds) != 0 {
		t.Fatalf("Fetch = %+v, want no seeds", seeds)
	}
	if observer.unreachable != 1 {
		t.Fatalf("SeedlistUnreachable reported %d times, want once", observer.unreachable)
	}
}
