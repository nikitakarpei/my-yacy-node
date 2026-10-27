package pagecopies_test

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

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/pagecopies"
)

const (
	retentionInTest = 15 * time.Second
	unboundedBytes  = 1 << 30
	pageKey         = "yacypeer\nhttps://example.org/"
	otherPageKey    = "yacypeer\nhttps://example.org/other"
	stopDeadline    = 10 * time.Second
)

var copiesStart = time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)

type steppedClock struct {
	now    time.Time
	asleep chan struct{}
	wake   chan struct{}
	mu     sync.Mutex
}

func newSteppedClock() *steppedClock {
	return &steppedClock{now: copiesStart, asleep: make(chan struct{}), wake: make(chan struct{})}
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

func openCopies(
	t *testing.T,
	directory string,
	maxBytes int64,
	clock *steppedClock,
) *pagecopies.Copies {
	t.Helper()
	copies, err := pagecopies.Open(t.Context(), directory, retentionInTest, maxBytes, clock)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return copies
}

func runCopies(t *testing.T, copies *pagecopies.Copies, clock *steppedClock) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- copies.Run(ctx) }()
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

func pageCopyOf(body string) pagecopies.PageCopy {
	return pagecopies.PageCopy{
		Header:     http.Header{"Content-Type": {"text/html"}},
		Body:       []byte(body),
		StatusCode: http.StatusOK,
	}
}

func assertCopyOf(t *testing.T, copies *pagecopies.Copies, key string, wantBody string) {
	t.Helper()
	pageCopy, found := copies.CopyFor(t.Context(), key)
	if !found {
		t.Fatalf("CopyFor(%q) found nothing, want a copy", key)
	}
	if !bytes.Equal(pageCopy.Body, []byte(wantBody)) || pageCopy.StatusCode != http.StatusOK ||
		pageCopy.Header.Get("Content-Type") != "text/html" {
		t.Fatalf("CopyFor(%q) = %+v, want the kept copy of %q", key, pageCopy, wantBody)
	}
}

func assertNoCopyOf(t *testing.T, copies *pagecopies.Copies, key string) {
	t.Helper()
	if _, found := copies.CopyFor(t.Context(), key); found {
		t.Fatalf("CopyFor(%q) found a copy, want none", key)
	}
}

func TestCopyForGivesBackTheKeptCopy(t *testing.T) {
	copies := openCopies(t, t.TempDir(), unboundedBytes, newSteppedClock())

	copies.Keep(t.Context(), pageKey, pageCopyOf("<p>page</p>"))

	assertCopyOf(t, copies, pageKey, "<p>page</p>")
}

func TestCopyForFindsNothingForAPageNeverKept(t *testing.T) {
	copies := openCopies(t, t.TempDir(), unboundedBytes, newSteppedClock())

	copies.Keep(t.Context(), pageKey, pageCopyOf("<p>page</p>"))

	assertNoCopyOf(t, copies, otherPageKey)
}

func TestKeepReplacesTheEarlierCopyOfAPage(t *testing.T) {
	copies := openCopies(t, t.TempDir(), unboundedBytes, newSteppedClock())

	copies.Keep(t.Context(), pageKey, pageCopyOf("<p>first</p>"))
	copies.Keep(t.Context(), pageKey, pageCopyOf("<p>second</p>"))

	assertCopyOf(t, copies, pageKey, "<p>second</p>")
}

func TestCopyForFindsNothingInAnUnreadableCopy(t *testing.T) {
	directory := t.TempDir()
	copies := openCopies(t, directory, unboundedBytes, newSteppedClock())
	copies.Keep(t.Context(), pageKey, pageCopyOf("<p>page</p>"))

	spoilEveryFileIn(t, directory)

	assertNoCopyOf(t, copies, pageKey)
}

func spoilEveryFileIn(t *testing.T, directory string) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("read the copy directory: %v", err)
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

func TestKeepLeavesNoCopyWhenTheDirectoryIsGone(t *testing.T) {
	directory := t.TempDir()
	copies := openCopies(t, directory, unboundedBytes, newSteppedClock())
	if err := os.RemoveAll(directory); err != nil {
		t.Fatalf("remove the copy directory: %v", err)
	}

	copies.Keep(t.Context(), pageKey, pageCopyOf("<p>page</p>"))

	assertNoCopyOf(t, copies, pageKey)
}

func TestOpenRefusesADirectoryItCannotCreate(t *testing.T) {
	blockingFile := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blockingFile, nil, 0o600); err != nil {
		t.Fatalf("create a file: %v", err)
	}

	_, err := pagecopies.Open(
		t.Context(),
		filepath.Join(blockingFile, "copies"),
		retentionInTest,
		unboundedBytes,
		newSteppedClock(),
	)
	if err == nil {
		t.Fatal("Open accepted a directory under a file")
	}
}

func TestRunForgetsACopyOnlyOnceItsRetentionPassed(t *testing.T) {
	clock := newSteppedClock()
	copies := openCopies(t, t.TempDir(), unboundedBytes, clock)
	copies.Keep(t.Context(), pageKey, pageCopyOf("<p>page</p>"))
	runCopies(t, copies, clock)

	sweepOnce(clock)
	assertCopyOf(t, copies, pageKey, "<p>page</p>")

	sweepOnce(clock)
	assertNoCopyOf(t, copies, pageKey)
}

func TestRunSurvivesACopyDirectoryThatIsGone(t *testing.T) {
	clock := newSteppedClock()
	directory := t.TempDir()
	copies := openCopies(t, directory, unboundedBytes, clock)
	runCopies(t, copies, clock)
	if err := os.RemoveAll(directory); err != nil {
		t.Fatalf("remove the copy directory: %v", err)
	}

	sweepOnce(clock)
}

func TestOpenForgetsCopiesKeptBeforeTheRetention(t *testing.T) {
	directory := t.TempDir()
	clock := newSteppedClock()
	openCopies(
		t,
		directory,
		unboundedBytes,
		clock,
	).Keep(t.Context(), pageKey, pageCopyOf("<p>page</p>"))

	clock.advance(retentionInTest + time.Second)
	reopened := openCopies(t, directory, unboundedBytes, clock)

	assertNoCopyOf(t, reopened, pageKey)
}

func TestOpenEvictsTheOldestCopiesOverTheMaximum(t *testing.T) {
	directory := t.TempDir()
	clock := newSteppedClock()
	copies := openCopies(t, directory, unboundedBytes, clock)
	copies.Keep(t.Context(), pageKey, pageCopyOf("<p>older</p>"))
	clock.advance(time.Second)
	copies.Keep(t.Context(), otherPageKey, pageCopyOf("<p>newer</p>"))

	reopened := openCopies(t, directory, sizeOfOneCopyIn(t, directory), clock)

	assertNoCopyOf(t, reopened, pageKey)
	assertCopyOf(t, reopened, otherPageKey, "<p>newer</p>")
}

func sizeOfOneCopyIn(t *testing.T, directory string) int64 {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) == 0 {
		t.Fatalf("read the copy directory: %v, %d entries", err, len(entries))
	}
	info, err := entries[0].Info()
	if err != nil {
		t.Fatalf("stat a copy: %v", err)
	}
	return info.Size()
}
