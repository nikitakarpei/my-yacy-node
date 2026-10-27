// Package cachedpages keeps the answers that origins give to page fetches, as files, for one
// retention and within a byte maximum, so that another fetch of the same page needs no origin
// fetch.
package cachedpages

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
	Name                  = "cachedpages"
	pageFileSuffix        = ".page"
	sweepsPerRetention    = 8
	shortestSweepInterval = 10 * time.Second
)

type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) error
}

type Page struct {
	Header     http.Header
	Body       []byte
	StatusCode int
}

type Pages struct {
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
) (*Pages, error) {
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return nil, fmt.Errorf("cached pages: %w", err)
	}
	cachedPages := &Pages{
		clock:     clock,
		directory: directory,
		retention: retention,
		maxBytes:  maxBytes,
	}
	cachedPages.sweep(ctx)
	return cachedPages, nil
}

type pageFile struct {
	modifiedAt time.Time
	path       string
	size       int64
}

func (c *Pages) sweep(ctx context.Context) {
	pageFiles, err := c.pageFiles(ctx)
	if err != nil {
		slog.WarnContext(ctx, "cached pages not swept", slog.Any("error", err))
		return
	}
	liveFiles, expiredFiles := splitAtCutoff(pageFiles, c.clock.Now().Add(-c.retention))
	keptFiles, evictedFiles := splitWithinMaxBytes(liveFiles, c.maxBytes)
	removeEvery(ctx, expiredFiles)
	removeEvery(ctx, evictedFiles)
	logSweep(ctx, keptFiles, len(expiredFiles), len(evictedFiles))
}

func (c *Pages) pageFiles(ctx context.Context) ([]pageFile, error) {
	entries, err := os.ReadDir(c.directory)
	if err != nil {
		return nil, fmt.Errorf("list cached pages: %w", err)
	}
	pageFiles := make([]pageFile, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			slog.WarnContext(ctx, "cached page not listed", slog.Any("error", err))
			continue
		}
		pageFiles = append(pageFiles, pageFile{
			modifiedAt: info.ModTime(),
			path:       filepath.Join(c.directory, entry.Name()),
			size:       info.Size(),
		})
	}
	return pageFiles, nil
}

func splitAtCutoff(
	pageFiles []pageFile,
	cutoff time.Time,
) (liveFiles, expiredFiles []pageFile) {
	for _, file := range pageFiles {
		if file.modifiedAt.Before(cutoff) {
			expiredFiles = append(expiredFiles, file)
			continue
		}
		liveFiles = append(liveFiles, file)
	}
	return liveFiles, expiredFiles
}

func splitWithinMaxBytes(
	liveFiles []pageFile,
	maxBytes int64,
) (keptFiles, evictedFiles []pageFile) {
	newestFirst := slices.SortedFunc(slices.Values(liveFiles), func(a, b pageFile) int {
		return b.modifiedAt.Compare(a.modifiedAt)
	})
	var sumOfCachedBytes int64
	for index, file := range newestFirst {
		if sumOfCachedBytes+file.size > maxBytes {
			return newestFirst[:index], newestFirst[index:]
		}
		sumOfCachedBytes += file.size
	}
	return newestFirst, nil
}

func removeEvery(ctx context.Context, pageFiles []pageFile) {
	for _, file := range pageFiles {
		if err := os.Remove(file.path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			slog.WarnContext(ctx, "cached page not removed", slog.Any("error", err))
		}
	}
}

func logSweep(
	ctx context.Context,
	keptFiles []pageFile,
	amountOfExpiredPages int,
	amountOfEvictedPages int,
) {
	var sumOfCachedBytes int64
	for _, file := range keptFiles {
		sumOfCachedBytes += file.size
	}
	slog.DebugContext(ctx, "cached pages swept",
		slog.Int("amountOfCachedPages", len(keptFiles)),
		slog.Int64("sumOfCachedBytes", sumOfCachedBytes),
		slog.Int("amountOfExpiredPages", amountOfExpiredPages),
		slog.Int("amountOfEvictedPages", amountOfEvictedPages),
	)
}

func (c *Pages) PageFor(ctx context.Context, key string) (Page, bool) {
	page, err := pageFrom(c.pathFor(key))
	if errors.Is(err, fs.ErrNotExist) {
		return Page{}, false
	}
	if err != nil {
		slog.WarnContext(ctx, "cached page unreadable", slog.Any("error", err))
		return Page{}, false
	}
	return page, true
}

func pageFrom(path string) (Page, error) {
	file, err := os.Open(path) //nolint:gosec // G304: the path is a hash under the cache directory.
	if err != nil {
		return Page{}, fmt.Errorf("open cached page: %w", err)
	}
	defer func() { _ = file.Close() }()
	var page Page
	if err := gob.NewDecoder(file).Decode(&page); err != nil {
		return Page{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return page, nil
}

func (c *Pages) pathFor(key string) string {
	sum := sha256.Sum256([]byte(key))
	return filepath.Join(c.directory, hex.EncodeToString(sum[:])+pageFileSuffix)
}

func (c *Pages) Keep(ctx context.Context, key string, page Page) {
	if err := c.write(key, page); err != nil {
		slog.WarnContext(ctx, "cached page not kept", slog.Any("error", err))
	}
}

func (c *Pages) write(key string, page Page) error {
	var encoded bytes.Buffer
	if err := gob.NewEncoder(&encoded).Encode(page); err != nil {
		return fmt.Errorf("encode cached page: %w", err)
	}
	path := c.pathFor(key)
	if err := os.WriteFile(path, encoded.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write cached page: %w", err)
	}
	keptAt := c.clock.Now()
	if err := os.Chtimes(path, keptAt, keptAt); err != nil {
		return fmt.Errorf("date cached page: %w", err)
	}
	return nil
}

func (c *Pages) Run(ctx context.Context) error {
	sweepInterval := max(c.retention/sweepsPerRetention, shortestSweepInterval)
	for c.clock.Sleep(ctx, sweepInterval) == nil {
		c.sweep(ctx)
	}
	return nil
}
