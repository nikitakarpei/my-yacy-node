// Package urlreferences knows, for every url, the words that reference it, so
// eviction can drop a url's postings without scanning the word index. It is a
// projection of the postings this node holds, and vault keeps it in step with
// them.
package urlreferences

import (
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type ReferenceQuery interface {
	WordsReferencing(tx *vault.Txn, url yacymodel.URLHash) ([]yacymodel.Hash, error)
}

func Open(v *vault.Vault) (ReferenceQuery, vault.Projection[yacymodel.RWIPosting], error) {
	words, err := registerWordsByURL(v)
	if err != nil {
		return nil, nil, err
	}

	return &urlReferences{words: words}, words, nil
}
