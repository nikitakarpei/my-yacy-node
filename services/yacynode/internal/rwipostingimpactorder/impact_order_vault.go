package rwipostingimpactorder

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const impactOrderBucket vault.Name = "rwi_impact"

func registerImpactOrder(v *vault.Vault) (*vault.Set[postingByImpact], error) {
	postings, err := v.RegisterSet(impactOrderBucket, postingByImpactKeyLayout)
	if err != nil {
		return nil, fmt.Errorf("register postings by impact: %w", err)
	}

	return postings, nil
}

var postingByImpactKeyParts = vault.TripleKey(
	hashkeypart.Hash,
	vault.IntegerKeyPartDescending,
	hashkeypart.URLHash,
)

var postingByImpactKeyLayout = postingByImpactKeyParts.KeyLayoutFor(
	func(entry postingByImpact) (yacymodel.Hash, int64, yacymodel.URLHash) {
		return entry.word, int64(entry.impact), entry.document
	},
	func(word yacymodel.Hash, impact int64, document yacymodel.URLHash) postingByImpact {
		return postingByImpact{word: word, impact: Impact(impact), document: document}
	},
)

func everyPostingOf(word yacymodel.Hash) vault.KeyRange {
	return postingByImpactKeyParts.KeysWithFirst(word)
}
