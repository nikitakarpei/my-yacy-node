// Package bleve keeps the search index of the text corpus in Bleve indexes in a directory
// of the local filesystem: one index per configured language and one for all other pages.
// It writes documents into them and answers searches of all of them at once.
package bleve

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	blevesearch "github.com/blevesearch/bleve/v2"

	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/languageindex"
)

const directoryPermission = 0o750

type Indexes struct {
	directory       string
	languageIndexes languageindex.LanguageIndexes
	openIndexes     map[string]blevesearch.Index
	alias           blevesearch.IndexAlias
}

func NewIndexes(directory string, languageIndexes languageindex.LanguageIndexes) *Indexes {
	return &Indexes{directory: directory, languageIndexes: languageIndexes}
}

func (indexes *Indexes) Bootstrap(ctx context.Context) error {
	if err := os.MkdirAll(indexes.directory, directoryPermission); err != nil {
		return fmt.Errorf("create search index directory %s: %w", indexes.directory, err)
	}
	if err := snippetHighlighterDefined(); err != nil {
		return err
	}
	openIndexes := make(map[string]blevesearch.Index, len(indexes.languageIndexes.All()))
	for _, languageIndex := range indexes.languageIndexes.All() {
		opened, err := indexes.openedOrCreated(ctx, languageIndex)
		if err != nil {
			return errors.Join(err, closed(openIndexes))
		}
		openIndexes[languageIndex.Name] = opened
	}
	indexes.openIndexes = openIndexes
	indexes.alias = blevesearch.NewIndexAlias(valuesOf(openIndexes)...)
	return nil
}

func (indexes *Indexes) openedOrCreated(
	ctx context.Context,
	languageIndex languageindex.LanguageIndex,
) (blevesearch.Index, error) {
	path := filepath.Join(indexes.directory, languageIndex.Name)
	opened, err := blevesearch.Open(path)
	if err == nil {
		slog.DebugContext(ctx, "bleve index present", slog.String("index", languageIndex.Name))
		return opened, nil
	}
	if !errors.Is(err, blevesearch.ErrorIndexPathDoesNotExist) {
		return nil, fmt.Errorf("open index %s: %w", languageIndex.Name, err)
	}
	indexMapping, err := indexMappingFor(languageIndex.Language)
	if err != nil {
		return nil, fmt.Errorf("map index %s: %w", languageIndex.Name, err)
	}
	created, err := blevesearch.New(path, indexMapping)
	if err != nil {
		return nil, fmt.Errorf("create index %s: %w", languageIndex.Name, err)
	}
	slog.DebugContext(ctx, "bleve index created", slog.String("index", languageIndex.Name))
	return created, nil
}

func valuesOf(openIndexes map[string]blevesearch.Index) []blevesearch.Index {
	values := make([]blevesearch.Index, 0, len(openIndexes))
	for _, opened := range openIndexes {
		values = append(values, opened)
	}
	return values
}

func (indexes *Indexes) Close() error {
	return closed(indexes.openIndexes)
}

func closed(openIndexes map[string]blevesearch.Index) error {
	var failures error
	for name, opened := range openIndexes {
		if err := opened.Close(); err != nil {
			failures = errors.Join(failures, fmt.Errorf("close index %s: %w", name, err))
		}
	}
	return failures
}

func (indexes *Indexes) indexFor(language string) blevesearch.Index {
	return indexes.openIndexes[indexes.languageIndexes.NameFor(language)]
}

func (indexes *Indexes) everyIndex() blevesearch.IndexAlias {
	return indexes.alias
}
