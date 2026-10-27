package cachedpages_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/cachedpages"
)

const (
	retentionInTest = 15 * time.Second
	unboundedBytes  = 1 << 30
	pageKey         = "yacypeer\nhttps://example.org/"
	otherPageKey    = "yacypeer\nhttps://example.org/other"
	stopDeadline    = 10 * time.Second
)

var cacheStart = time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)

type steppedClock struct {
	now    time.Time
	asleep chan struct{}
	wake   chan struct{}
	mu     sync.Mutex
}

func newSteppedClock() *steppedClock {
	return &steppedClock{now: cacheStart, asleep: make(chan struct{}), wake: make(chan struct{})}
}

func (c *steppedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *steppedClock) Sleep(ctx context.Context, d time.Duration) error {
	select {
	case c.asleep <- struct{}{}:
	case <-ctx.Done():
		return fmt.Errorf("sleep: %w", ctx.Err())
	}
	select {
	case <-c.wake:
	case <-ctx.Done():
		return fmt.Errorf("sleep: %w", ctx.Err())
	}
	c.advance(d)
	return nil
}

func (c *steppedClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func openCachedPages(
	t *testing.T,
	directory string,
	maxBytes int64,
	clock *steppedClock,
) *cachedpages.Pages {
	t.Helper()
	cachedPages, err := cachedpages.Open(t.Context(), directory, retentionInTest, maxBytes, clock)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return cachedPages
}

func runCachedPages(t *testing.T, cachedPages *cachedpages.Pages, clock *steppedClock) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- cachedPages.Run(ctx) }()
	<-clock.asleep
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-runDone:
			if err != nil {
				t.Errorf("Run: %v", err)
			}
		case <-time.After(stopDeadline):
			t.Error("Run did not stop after cancel")
		}
	})
}

func sweepOnce(clock *steppedClock) {
	clock.wake <- struct{}{}
	<-clock.asleep
}

func pageOf(body string) cachedpages.Page {
	return cachedpages.Page{
		Header:     http.Header{"Content-Type": {"text/html"}},
		Body:       []byte(body),
		StatusCode: http.StatusOK,
	}
}

func assertCachedPage(t *testing.T, cachedPages *cachedpages.Pages, key string, wantBody string) {
	t.Helper()
	cachedPage, found := cachedPages.PageFor(t.Context(), key)
	if !found {
		t.Fatalf("PageFor(%q) found nothing, want a cached page", key)
	}
	if !bytes.Equal(cachedPage.Body, []byte(wantBody)) || cachedPage.StatusCode != http.StatusOK ||
		cachedPage.Header.Get("Content-Type") != "text/html" {
		t.Fatalf("PageFor(%q) = %+v, want the cached page of %q", key, cachedPage, wantBody)
	}
}

func assertNoCachedPage(t *testing.T, cachedPages *cachedpages.Pages, key string) {
	t.Helper()
	if _, found := cachedPages.PageFor(t.Context(), key); found {
		t.Fatalf("PageFor(%q) found a cached page, want none", key)
	}
}

func TestPageForGivesBackTheKeptPage(t *testing.T) {
	cachedPages := openCachedPages(t, t.TempDir(), unboundedBytes, newSteppedClock())

	cachedPages.Keep(t.Context(), pageKey, pageOf("<p>page</p>"))

	assertCachedPage(t, cachedPages, pageKey, "<p>page</p>")
}

func TestPageForFindsNothingForAPageNeverKept(t *testing.T) {
	cachedPages := openCachedPages(t, t.TempDir(), unboundedBytes, newSteppedClock())

	cachedPages.Keep(t.Context(), pageKey, pageOf("<p>page</p>"))

	assertNoCachedPage(t, cachedPages, otherPageKey)
}

func TestKeepReplacesTheEarlierCachedPage(t *testing.T) {
	cachedPages := openCachedPages(t, t.TempDir(), unboundedBytes, newSteppedClock())

	cachedPages.Keep(t.Context(), pageKey, pageOf("<p>first</p>"))
	cachedPages.Keep(t.Context(), pageKey, pageOf("<p>second</p>"))

	assertCachedPage(t, cachedPages, pageKey, "<p>second</p>")
}

func TestPageForFindsNothingInAnUnreadableFile(t *testing.T) {
	directory := t.TempDir()
	cachedPages := openCachedPages(t, directory, unboundedBytes, newSteppedClock())
	cachedPages.Keep(t.Context(), pageKey, pageOf("<p>page</p>"))

	spoilEveryFileIn(t, directory)

	assertNoCachedPage(t, cachedPages, pageKey)
}

func spoilEveryFileIn(t *testing.T, directory string) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read the cache directory: %v", err)
	}
	for _, entry := range entries {
		if err := os.WriteFile(
			filepath.Join(directory, entry.Name()),
			[]byte("spoilt"),
			0o600,
		); err != nil {
			t.Fatalf("spoil %s: %v", entry.Name(), err)
		}
	}
}

func TestKeepCachesNothingWhenTheDirectoryIsGone(t *testing.T) {
	directory := t.TempDir()
	cachedPages := openCachedPages(t, directory, unboundedBytes, newSteppedClock())
	if err := os.RemoveAll(directory); err != nil {
		t.Fatalf("remove the cache directory: %v", err)
	}

	cachedPages.Keep(t.Context(), pageKey, pageOf("<p>page</p>"))

	assertNoCachedPage(t, cachedPages, pageKey)
}

func TestOpenRefusesADirectoryItCannotCreate(t *testing.T) {
	blockingFile := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blockingFile, nil, 0o600); err != nil {
		t.Fatalf("create a file: %v", err)
	}

	_, err := cachedpages.Open(
		t.Context(),
		filepath.Join(blockingFile, "cachedPages"),
		retentionInTest,
		unboundedBytes,
		newSteppedClock(),
	)
	if err == nil {
		t.Fatal("Open accepted a directory under a file")
	}
}

func TestRunForgetsACachedPageOnlyOnceItsRetentionPassed(t *testing.T) {
	clock := newSteppedClock()
	cachedPages := openCachedPages(t, t.TempDir(), unboundedBytes, clock)
	cachedPages.Keep(t.Context(), pageKey, pageOf("<p>page</p>"))
	runCachedPages(t, cachedPages, clock)

	sweepOnce(clock)
	assertCachedPage(t, cachedPages, pageKey, "<p>page</p>")

	sweepOnce(clock)
	assertNoCachedPage(t, cachedPages, pageKey)
}

func TestRunSurvivesACacheDirectoryThatIsGone(t *testing.T) {
	clock := newSteppedClock()
	directory := t.TempDir()
	cachedPages := openCachedPages(t, directory, unboundedBytes, clock)
	runCachedPages(t, cachedPages, clock)
	if err := os.RemoveAll(directory); err != nil {
		t.Fatalf("remove the cache directory: %v", err)
	}

	sweepOnce(clock)
}

func TestOpenForgetsPagesCachedBeforeTheRetention(t *testing.T) {
	directory := t.TempDir()
	clock := newSteppedClock()
	openCachedPages(
		t,
		directory,
		unboundedBytes,
		clock,
	).Keep(t.Context(), pageKey, pageOf("<p>page</p>"))

	clock.advance(retentionInTest + time.Second)
	reopened := openCachedPages(t, directory, unboundedBytes, clock)

	assertNoCachedPage(t, reopened, pageKey)
}

func TestOpenEvictsTheOldestPagesOverTheMaximum(t *testing.T) {
	directory := t.TempDir()
	clock := newSteppedClock()
	cachedPages := openCachedPages(t, directory, unboundedBytes, clock)
	cachedPages.Keep(t.Context(), pageKey, pageOf("<p>older</p>"))
	clock.advance(time.Second)
	cachedPages.Keep(t.Context(), otherPageKey, pageOf("<p>newer</p>"))

	reopened := openCachedPages(t, directory, sizeOfOneCachedPageIn(t, directory), clock)

	assertNoCachedPage(t, reopened, pageKey)
	assertCachedPage(t, reopened, otherPageKey, "<p>newer</p>")
}

func sizeOfOneCachedPageIn(t *testing.T, directory string) int64 {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) == 0 {
		t.Fatalf("read the cache directory: %v, %d entries", err, len(entries))
	}
	info, err := entries[0].Info()
	if err != nil {
		t.Fatalf("stat a cached page: %v", err)
	}
	return info.Size()
}
