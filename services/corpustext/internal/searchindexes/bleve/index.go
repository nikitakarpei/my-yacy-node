package bleve

import (
	"context"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
)

type Index struct {
	indexes *Indexes
}

func NewIndex(indexes *Indexes) *Index {
	return &Index{indexes: indexes}
}

func (idx *Index) Index(_ context.Context, document searchdocument.Document) error {
	if err := idx.indexes.indexFor(document.Language).Index(document.URL, document); err != nil {
		return fmt.Errorf("index document %s: %w", document.URL, err)
	}
	return nil
}
