// Package filesystem holds each crawled page's markdown in a file under a directory of the
// local filesystem and yields it back by canonical URL, with the time it was written and its
// version.
package filesystem

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecall"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore"
)

const (
	directoryPermission = 0o750
	filePermission      = 0o600
	shardNameLength     = 2
	pendingNameInfix    = ".pending-"
)

type Clock interface {
	Now() time.Time
}

type Corpus struct {
	root  *os.Root
	clock Clock
}

func OpenCorpus(directory string, clock Clock) (*Corpus, error) {
	if err := os.MkdirAll(directory, directoryPermission); err != nil {
		return nil, fmt.Errorf("create page markdown directory %q: %w", directory, err)
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, fmt.Errorf("open page markdown directory %q: %w", directory, err)
	}
	return &Corpus{root: root, clock: clock}, nil
}

func (c *Corpus) Put(
	_ context.Context,
	canonicalURL canonicalurl.CanonicalURL,
	markdown []byte,
) error {
	if err := c.replace(fileNameOf(canonicalURL), markdown); err != nil {
		return fmt.Errorf("put markdown for %q: %w", canonicalURL, err)
	}
	return nil
}

func fileNameOf(canonicalURL canonicalurl.CanonicalURL) string {
	objectName := pagemarkdownstore.ObjectNameOf(canonicalURL)
	return filepath.Join(objectName[:shardNameLength], objectName)
}

func (c *Corpus) replace(fileName string, markdown []byte) error {
	if err := c.root.MkdirAll(filepath.Dir(fileName), directoryPermission); err != nil {
		return fmt.Errorf("create the shard directory: %w", err)
	}
	pendingName := fileName + pendingNameInfix + rand.Text()
	defer func() { _ = c.root.Remove(pendingName) }()
	if err := c.write(pendingName, markdown); err != nil {
		return err
	}
	storedAt := c.clock.Now()
	if err := c.root.Chtimes(pendingName, storedAt, storedAt); err != nil {
		return fmt.Errorf("record the time it was stored: %w", err)
	}
	if err := c.root.Rename(pendingName, fileName); err != nil {
		return fmt.Errorf("replace the earlier markdown: %w", err)
	}
	return nil
}

func (c *Corpus) write(pendingName string, markdown []byte) error {
	pending, err := c.root.OpenFile(
		pendingName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, filePermission,
	)
	if err != nil {
		return fmt.Errorf("create the pending file: %w", err)
	}
	_, writeErr := pending.Write(markdown)
	if err := errors.Join(writeErr, pending.Sync(), pending.Close()); err != nil {
		return fmt.Errorf("write the pending file: %w", err)
	}
	return nil
}

func (c *Corpus) MarkdownOf(
	_ context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) (markdownrecall.StoredMarkdown, bool, error) {
	file, err := c.root.Open(fileNameOf(canonicalURL))
	if errors.Is(err, fs.ErrNotExist) {
		return markdownrecall.StoredMarkdown{}, false, nil
	}
	if err != nil {
		return markdownrecall.StoredMarkdown{}, false,
			fmt.Errorf("open markdown for %q: %w", canonicalURL, err)
	}
	defer func() { _ = file.Close() }()

	stored, err := storedMarkdownFrom(file)
	if err != nil {
		return markdownrecall.StoredMarkdown{}, false,
			fmt.Errorf("read markdown for %q: %w", canonicalURL, err)
	}
	return stored, true, nil
}

func storedMarkdownFrom(file *os.File) (markdownrecall.StoredMarkdown, error) {
	status, err := file.Stat()
	if err != nil {
		return markdownrecall.StoredMarkdown{}, fmt.Errorf("read the time it was stored: %w", err)
	}
	markdown := make([]byte, status.Size())
	if _, err := io.ReadFull(file, markdown); err != nil {
		return markdownrecall.StoredMarkdown{}, fmt.Errorf("read the markdown body: %w", err)
	}
	return markdownrecall.StoredMarkdown{
		Markdown: markdown,
		StoredAt: status.ModTime().UTC(),
		Version:  markdownrecall.VersionOf(markdown),
	}, nil
}
