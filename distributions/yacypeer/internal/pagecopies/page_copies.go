// Package pagecopies keeps copies of the answers that origins give to page fetches, as files, for
// one retention and within a byte maximum, so that another fetch of the same page needs no origin
// fetch.
package pagecopies

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"time"
)

const (
	Name                  = "pagecopies"
	copyFileSuffix        = ".copy"
	sweepsPerRetention    = 8
	shortestSweepInterval = 10 * time.Second
)

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type PageCopy struct {
	Header     http.Header
	Body       []byte
	StatusCode int
}

type Copies struct {
	clock     Clock
	directory string
	retention time.Duration
	maxBytes  int64
}

func Open(
	ctx context.Context,
	directory string,
	retention time.Duration,
	maxBytes int64,
	clock Clock,
) (*Copies, error) {
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return nil, fmt.Errorf("page copies: %w", err)
	}
	copies := &Copies{clock: clock, directory: directory, retention: retention, maxBytes: maxBytes}
	copies.sweep(ctx)
	return copies, nil
}

type copyFile struct {
	modifiedAt time.Time
	path       string
	size       int64
}

func (c *Copies) sweep(ctx context.Context) {
	copyFiles, err := c.copyFiles(ctx)
	if err != nil {
		slog.WarnContext(ctx, "page copies not swept", slog.Any("error", err))
		return
	}
	liveFiles, expiredFiles := splitAtCutoff(copyFiles, c.clock.Now().Add(-c.retention))
	keptFiles, evictedFiles := splitWithinMaxBytes(liveFiles, c.maxBytes)
	removeEvery(ctx, expiredFiles)
	removeEvery(ctx, evictedFiles)
	logSweep(ctx, keptFiles, len(expiredFiles), len(evictedFiles))
}

func (c *Copies) copyFiles(ctx context.Context) ([]copyFile, error) {
	entries, err := os.ReadDir(c.directory)
	if err != nil {
		return nil, fmt.Errorf("list page copies: %w", err)
	}
	copyFiles := make([]copyFile, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			slog.WarnContext(ctx, "page copy not listed", slog.Any("error", err))
			continue
		}
		copyFiles = append(copyFiles, copyFile{
			modifiedAt: info.ModTime(),
			path:       filepath.Join(c.directory, entry.Name()),
			size:       info.Size(),
		})
	}
	return copyFiles, nil
}

func splitAtCutoff(copyFiles []copyFile, cutoff time.Time) (liveFiles, expiredFiles []copyFile) {
	for _, file := range copyFiles {
		if file.modifiedAt.Before(cutoff) {
			expiredFiles = append(expiredFiles, file)
			continue
		}
		liveFiles = append(liveFiles, file)
	}
	return liveFiles, expiredFiles
}

func splitWithinMaxBytes(
	liveFiles []copyFile,
	maxBytes int64,
) (keptFiles, evictedFiles []copyFile) {
	newestFirst := slices.SortedFunc(slices.Values(liveFiles), func(a, b copyFile) int {
		return b.modifiedAt.Compare(a.modifiedAt)
	})
	var sumOfCopyBytes int64
	for index, file := range newestFirst {
		if sumOfCopyBytes+file.size > maxBytes {
			return newestFirst[:index], newestFirst[index:]
		}
		sumOfCopyBytes += file.size
	}
	return newestFirst, nil
}

func removeEvery(ctx context.Context, copyFiles []copyFile) {
	for _, file := range copyFiles {
		if err := os.Remove(file.path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			slog.WarnContext(ctx, "page copy not removed", slog.Any("error", err))
		}
	}
}

func logSweep(
	ctx context.Context,
	keptFiles []copyFile,
	amountOfExpiredCopies int,
	amountOfEvictedCopies int,
) {
	var sumOfCopyBytes int64
	for _, file := range keptFiles {
		sumOfCopyBytes += file.size
	}
	slog.DebugContext(ctx, "page copies swept",
		slog.Int("amountOfCopies", len(keptFiles)),
		slog.Int64("sumOfCopyBytes", sumOfCopyBytes),
		slog.Int("amountOfExpiredCopies", amountOfExpiredCopies),
		slog.Int("amountOfEvictedCopies", amountOfEvictedCopies),
	)
}

func (c *Copies) CopyFor(ctx context.Context, key string) (PageCopy, bool) {
	pageCopy, err := copyFrom(c.pathFor(key))
	if errors.Is(err, fs.ErrNotExist) {
		return PageCopy{}, false
	}
	if err != nil {
		slog.WarnContext(ctx, "page copy unreadable", slog.Any("error", err))
		return PageCopy{}, false
	}
	return pageCopy, true
}

func copyFrom(path string) (PageCopy, error) {
	file, err := os.Open(path) //nolint:gosec // G304: the path is a hash under the copy directory.
	if err != nil {
		return PageCopy{}, fmt.Errorf("open page copy: %w", err)
	}
	defer func() { _ = file.Close() }()
	var pageCopy PageCopy
	if err := gob.NewDecoder(file).Decode(&pageCopy); err != nil {
		return PageCopy{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return pageCopy, nil
}

func (c *Copies) pathFor(key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(c.directory, hex.EncodeToString(sum[:])+copyFileSuffix)
}

func (c *Copies) Keep(ctx context.Context, key string, pageCopy PageCopy) {
	if err := c.write(key, pageCopy); err != nil {
		slog.WarnContext(ctx, "page copy not kept", slog.Any("error", err))
	}
}

func (c *Copies) write(key string, pageCopy PageCopy) error {
	var encoded bytes.Buffer
	if err := gob.NewEncoder(&encoded).Encode(pageCopy); err != nil {
		return fmt.Errorf("encode page copy: %w", err)
	}
	path := c.pathFor(key)
	if err := os.WriteFile(path, encoded.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write page copy: %w", err)
	}
	keptAt := c.clock.Now()
	if err := os.Chtimes(path, keptAt, keptAt); err != nil {
		return fmt.Errorf("date page copy: %w", err)
	}
	return nil
}

func (c *Copies) Run(ctx context.Context) error {
	sweepInterval := max(c.retention/sweepsPerRetention, shortestSweepInterval)
	for c.clock.Sleep(ctx, sweepInterval) == nil {
		c.sweep(ctx)
	}
	return nil
}
