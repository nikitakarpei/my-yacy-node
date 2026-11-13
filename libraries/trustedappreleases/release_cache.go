package trustedappreleases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type CacheObserver interface {
	CacheUnread(ctx context.Context, cause error)
	CacheUnwritten(ctx context.Context, cause error)
}

type CachedRefresh struct {
	DenialsReadAt time.Time        `json:"denialsReadAt"`
	Releases      []TrustedRelease `json:"releases"`
	Denials       []Denial         `json:"denials"`
}

type ReleaseCache struct {
	observer    CacheObserver
	path        string
	lastRefresh CachedRefresh
}

func OpenReleaseCache(ctx context.Context, path string, observer CacheObserver) *ReleaseCache {
	cache := &ReleaseCache{path: path, observer: observer}
	cache.readFile(ctx)

	return cache
}

func (c *ReleaseCache) readFile(ctx context.Context) {
	content, err := os.ReadFile(c.path)
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err == nil {
		err = json.Unmarshal(content, &c.lastRefresh)
	}
	if err != nil {
		c.lastRefresh = CachedRefresh{}
		c.observer.CacheUnread(ctx, fmt.Errorf("read the cache of trusted releases: %w", err))
	}
}

func (c *ReleaseCache) LastRefresh() CachedRefresh {
	return c.lastRefresh
}

func (c *ReleaseCache) Write(ctx context.Context, refresh CachedRefresh) {
	content, err := json.Marshal(refresh)
	if err == nil {
		err = writeReplacing(c.path, content)
	}
	if err != nil {
		c.observer.CacheUnwritten(ctx, err)

		return
	}
	c.lastRefresh = refresh
}

func writeReplacing(path string, content []byte) error {
	written, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("write the cache of trusted releases: %w", err)
	}
	defer func() { _ = os.Remove(written.Name()) }()
	_, err = written.Write(content)
	if closeErr := written.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(written.Name(), path)
	}
	if err != nil {
		return fmt.Errorf("write the cache of trusted releases: %w", err)
	}

	return nil
}
