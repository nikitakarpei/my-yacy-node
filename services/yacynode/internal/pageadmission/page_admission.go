package pageadmission

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagerwi"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type pageAdmission struct {
	vault    *vault.Vault
	urls     urlmeta.URLEvictor
	metadata urlmeta.URLMetadataWriter
	postings rwipostings.PostingWriter
	pause    time.Duration
}

func (a pageAdmission) Receive(
	ctx context.Context,
	pageRWI pagerwi.PageRWI,
) (Receipt, error) {
	atCapacity, err := a.vault.AtCapacity(ctx)
	if err != nil {
		return Receipt{}, fmt.Errorf("check capacity: %w", err)
	}
	if atCapacity {
		return Receipt{Busy: true, Pause: a.pause}, nil
	}

	err = a.vault.Update(ctx, func(tx *vault.Txn) error {
		return a.replaceEverythingHeldForPage(ctx, tx, pageRWI)
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
	pageRWI pagerwi.PageRWI,
) error {
	metadata := pageRWI.Metadata()
	if err := a.purge(ctx, tx, metadata.Hash); err != nil {
		return err
	}
	pagePostings, indexed := pageRWI.PagePostings().Get()
	if !indexed {
		return nil
	}
	if err := a.writeMetadata(tx, metadata); err != nil {
		return err
	}

	return a.writePostings(tx, pagePostings)
}

func (a pageAdmission) purge(ctx context.Context, tx *vault.Txn, url yacymodel.URLHash) error {
	if _, err := a.urls.Purge(ctx, tx, []yacymodel.URLHash{url}); err != nil {
		return fmt.Errorf("purge page url: %w", err)
	}

	return nil
}

func (a pageAdmission) writeMetadata(tx *vault.Txn, metadata yacymodel.URLMetadata) error {
	if err := a.metadata.Write(tx, metadata, pageversion.SourceThisNode()); err != nil {
		return fmt.Errorf("write page url metadata: %w", err)
	}

	return nil
}

func (a pageAdmission) writePostings(tx *vault.Txn, pagePostings yacymodel.PagePostings) error {
	if err := a.postings.Write(tx, pagePostings); err != nil {
		return fmt.Errorf("write page postings: %w", err)
	}

	return nil
}

var _ PageReceiver = pageAdmission{}
