package rwiimpactorder

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type postingByImpact struct {
	word     yacymodel.Hash
	impact   Impact
	document yacymodel.URLHash
}

func postingByImpactOf(posting yacymodel.RWIPosting) postingByImpact {
	return postingByImpact{
		word:     posting.WordHash,
		impact:   ImpactOf(posting),
		document: posting.URLHash,
	}
}

type impactOrder struct {
	postings *vault.Set[postingByImpact]
}

func openImpactOrder(v *vault.Vault) (*impactOrder, error) {
	postings, err := registerImpactOrder(v)
	if err != nil {
		return nil, err
	}

	return &impactOrder{postings: postings}, nil
}

func (o *impactOrder) PostingStored(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	if _, err := o.postings.Add(tx, postingByImpactOf(posting)); err != nil {
		return fmt.Errorf("record posting impact: %w", err)
	}

	return nil
}

func (o *impactOrder) PostingUpdated(
	tx *vault.Txn,
	previous yacymodel.RWIPosting,
	current yacymodel.RWIPosting,
) error {
	if ImpactOf(previous) == ImpactOf(current) {
		return nil
	}
	if err := o.PostingPurged(tx, previous); err != nil {
		return err
	}

	return o.PostingStored(tx, current)
}

func (o *impactOrder) PostingPurged(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	if _, err := o.postings.Remove(tx, postingByImpactOf(posting)); err != nil {
		return fmt.Errorf("drop posting impact: %w", err)
	}

	return nil
}

func (o *impactOrder) ScanPostingsInImpactOrder(
	tx *vault.Txn,
	word yacymodel.Hash,
	visit func(document yacymodel.URLHash, impact Impact) (bool, error),
) error {
	if err := o.postings.Scan(
		tx,
		everyPostingOf(word),
		func(entry postingByImpact) (bool, error) {
			return visit(entry.document, entry.impact)
		},
	); err != nil {
		return fmt.Errorf("scan postings in impact order: %w", err)
	}

	return nil
}

func (o *impactOrder) LargestImpactOf(
	tx *vault.Txn,
	word yacymodel.Hash,
) (Impact, bool, error) {
	var (
		largestImpact Impact
		found         bool
	)
	if err := o.ScanPostingsInImpactOrder(
		tx,
		word,
		func(_ yacymodel.URLHash, impact Impact) (bool, error) {
			largestImpact, found = impact, true

			return false, nil
		},
	); err != nil {
		return 0, false, err
	}

	return largestImpact, found, nil
}

var _ ImpactOrderProjection = (*impactOrder)(nil)
