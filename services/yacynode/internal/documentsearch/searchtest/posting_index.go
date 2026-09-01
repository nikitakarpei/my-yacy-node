// Package searchtest holds the in-memory fakes the documentsearch tests share:
// a posting index, a URL directory, their failing variants, and deterministic
// hashes derived from short labels.
package searchtest

import (
	"context"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PostingIndex struct {
	Postings map[yacymodel.Hash][]yacymodel.RWIPosting
}

func (s PostingIndex) RWICount(*vault.Txn) (int, error) {
	return len(s.Postings), nil
}

func (s PostingIndex) ScanWord(
	_ context.Context,
	_ *vault.Txn,
	word yacymodel.Hash,
	visit func(yacymodel.RWIPosting) (bool, error),
) error {
	for _, entry := range s.Postings[word] {
		entry.WordHash = word
		keepGoing, err := visit(entry)
		if err != nil {
			return err
		}
		if !keepGoing {
			return nil
		}
	}

	return nil
}

func (s PostingIndex) PostingOf(
	_ *vault.Txn,
	word yacymodel.Hash,
	url yacymodel.URLHash,
) (yacymodel.RWIPosting, bool, error) {
	for _, entry := range s.Postings[word] {
		if entry.URLHash == url {
			entry.WordHash = word

			return entry, true, nil
		}
	}

	return yacymodel.RWIPosting{}, false, nil
}

type FailingPostingIndex struct {
	Err error
}

func (s FailingPostingIndex) RWICount(*vault.Txn) (int, error) {
	return 0, s.Err
}

func (s FailingPostingIndex) ScanWord(
	context.Context,
	*vault.Txn,
	yacymodel.Hash,
	func(yacymodel.RWIPosting) (bool, error),
) error {
	return s.Err
}

func (s FailingPostingIndex) PostingOf(
	*vault.Txn,
	yacymodel.Hash,
	yacymodel.URLHash,
) (yacymodel.RWIPosting, bool, error) {
	return yacymodel.RWIPosting{}, false, s.Err
}
