package pageadmission

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagerwi"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type pageAdmission struct {
	vault     *vault.Vault
	urls      urlmeta.URLEvictor
	metadata  urlmeta.URLMetadataWriter
	postings  rwipostings.PostingWriter
	stopwords Stopwords
	pause     time.Duration
}

func (a pageAdmission) Receive(ctx context.Context, page pagerwi.PageRWI) (Receipt, error) {
	atCapacity, err := a.vault.AtCapacity(ctx)
	if err != nil {
		return Receipt{}, fmt.Errorf("check capacity: %w", err)
	}
	if atCapacity {
		return Receipt{Busy: true, Pause: a.pause}, nil
	}

	err = a.vault.Update(ctx, func(tx *vault.Txn) error {
		return a.replaceEverythingHeldForPage(ctx, tx, page)
	})
	if errors.Is(err, vault.ErrAtCapacity) {
		return Receipt{Busy: true, Pause: a.pause}, nil
	}
	if err != nil {
		return Receipt{}, fmt.Errorf("receive page rwi: %w", err)
	}

	return Receipt{}, nil
}

func (a pageAdmission) replaceEverythingHeldForPage(
	ctx context.Context,
	tx *vault.Txn,
	page pagerwi.PageRWI,
) error {
	if _, err := a.urls.Purge(
		ctx,
		tx,
		[]yacymodel.URLHash{page.Metadata.Hash},
	); err != nil {
		return fmt.Errorf("purge page url: %w", err)
	}
	if _, err := a.metadata.Write(tx, page.Metadata, urlmeta.SourceThisNode()); err != nil {
		return fmt.Errorf("write page url metadata: %w", err)
	}

	return a.postings.Write(tx, a.postingsWithoutStopwordsOf(page.Postings)...)
}

func (a pageAdmission) postingsWithoutStopwordsOf(
	postings []yacymodel.RWIPosting,
) []yacymodel.RWIPosting {
	postingsWithoutStopwords := make([]yacymodel.RWIPosting, 0, len(postings))
	for _, posting := range postings {
		if !a.stopwords.Contains(posting.WordHash, posting.Language) {
			postingsWithoutStopwords = append(postingsWithoutStopwords, posting)
		}
	}

	return postingsWithoutStopwords
}

var _ PageReceiver = pageAdmission{}
