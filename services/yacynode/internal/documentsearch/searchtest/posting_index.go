// Package searchtest holds the in-memory fakes the documentsearch tests share:
// a posting index that answers by word and document, in impact order and by
// amount, a URL directory, their failing variants, and deterministic hashes
// derived from short labels.
package searchtest

import (
	"cmp"
	"slices"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostingimpactorder"
)

type PostingIndex struct {
	Postings map[yacymodel.Hash][]yacymodel.RWIPosting
}

func (s PostingIndex) RWICount(*vault.Txn) (int, error) {
	return len(s.Postings), nil
}

func (s PostingIndex) PostingOf(
	_ *vault.Txn,
	word yacymodel.Hash,
	document yacymodel.URLHash,
) (yacymodel.RWIPosting, bool, error) {
	for _, entry := range s.Postings[word] {
		if entry.URLHash == document {
			entry.WordHash = word

			return entry, true, nil
		}
	}

	return yacymodel.RWIPosting{}, false, nil
}

func (s PostingIndex) AmountOfPostingsOf(
	_ *vault.Txn,
	word yacymodel.Hash,
) (int, error) {
	return len(s.Postings[word]), nil
}

func (s PostingIndex) ScanPostingsInImpactOrder(
	_ *vault.Txn,
	word yacymodel.Hash,
	visit func(document yacymodel.URLHash, impact rwipostingimpactorder.Impact) (bool, error),
) error {
	for _, entry := range s.postingsInImpactOrder(word) {
		keepGoing, err := visit(entry.URLHash, rwipostingimpactorder.ImpactOf(entry))
		if err != nil {
			return err
		}
		if !keepGoing {
			return nil
		}
	}

	return nil
}

func (s PostingIndex) postingsInImpactOrder(
	word yacymodel.Hash,
) []yacymodel.RWIPosting {
	ordered := slices.Clone(s.Postings[word])
	slices.SortStableFunc(ordered, func(a, b yacymodel.RWIPosting) int {
		return cmp.Or(
			cmp.Compare(rwipostingimpactorder.ImpactOf(b), rwipostingimpactorder.ImpactOf(a)),
			yacymodel.CompareInAlphabetOrder(a.URLHash.String(), b.URLHash.String()),
		)
	})

	return ordered
}

func (s PostingIndex) LargestImpactOf(
	_ *vault.Txn,
	word yacymodel.Hash,
) (rwipostingimpactorder.Impact, bool, error) {
	ordered := s.postingsInImpactOrder(word)
	if len(ordered) == 0 {
		return 0, false, nil
	}

	return rwipostingimpactorder.ImpactOf(ordered[0]), true, nil
}

type FailingPostingIndex struct {
	Err error
}

func (s FailingPostingIndex) RWICount(*vault.Txn) (int, error) {
	return 0, s.Err
}

func (s FailingPostingIndex) PostingOf(
	*vault.Txn,
	yacymodel.Hash,
	yacymodel.URLHash,
) (yacymodel.RWIPosting, bool, error) {
	return yacymodel.RWIPosting{}, false, s.Err
}

func (s FailingPostingIndex) AmountOfPostingsOf(*vault.Txn, yacymodel.Hash) (int, error) {
	return 0, s.Err
}

func (s FailingPostingIndex) ScanPostingsInImpactOrder(
	*vault.Txn,
	yacymodel.Hash,
	func(yacymodel.URLHash, rwipostingimpactorder.Impact) (bool, error),
) error {
	return s.Err
}

func (s FailingPostingIndex) LargestImpactOf(
	*vault.Txn,
	yacymodel.Hash,
) (rwipostingimpactorder.Impact, bool, error) {
	return 0, false, s.Err
}
