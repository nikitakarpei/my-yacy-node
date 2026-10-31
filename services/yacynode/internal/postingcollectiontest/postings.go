// Package postingcollectiontest holds a plain vault collection of postings for
// the tests of the projections over the postings this node holds. It stores and
// removes whole postings, so a projection test needs no posting store.
package postingcollectiontest

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const postingsBucket vault.Name = "rwi"

type Postings struct {
	collection *vault.Collection[postingIdentity, yacymodel.RWIPosting]
}

func Open(
	ctx context.Context,
	v *vault.Vault,
	projections ...vault.Projection[yacymodel.RWIPosting],
) (*Postings, error) {
	collection, err := v.RegisterProjectedCollection(
		ctx,
		postingsBucket,
		postingIdentityKeyLayout,
		postingCodec{},
		vault.ProjectedAs(storedPosting, projections...),
	)
	if err != nil {
		return nil, fmt.Errorf("register posting collection: %w", err)
	}

	return &Postings{collection: collection}, nil
}

type postingIdentity struct {
	word yacymodel.Hash
	url  yacymodel.URLHash
}

var postingIdentityKeyLayout = vault.PairKey(
	vault.TextKeyPartFrom(yacymodel.Hash.String, yacymodel.ParseHash),
	vault.TextKeyPartFrom(yacymodel.URLHash.String, yacymodel.ParseURLHash),
).KeyLayoutFor(
	func(identity postingIdentity) (yacymodel.Hash, yacymodel.URLHash) {
		return identity.word, identity.url
	},
	func(word yacymodel.Hash, url yacymodel.URLHash) postingIdentity {
		return postingIdentity{word: word, url: url}
	},
)

type postingCodec struct{}

func (postingCodec) Encode(posting yacymodel.RWIPosting) ([]byte, error) {
	encoded, err := json.Marshal(posting)
	if err != nil {
		return nil, fmt.Errorf("encode posting: %w", err)
	}

	return encoded, nil
}

func (postingCodec) Decode(stored []byte) (yacymodel.RWIPosting, error) {
	var posting yacymodel.RWIPosting
	if err := json.Unmarshal(stored, &posting); err != nil {
		return yacymodel.RWIPosting{}, fmt.Errorf("decode posting: %w", err)
	}

	return posting, nil
}

func storedPosting(_ postingIdentity, posting yacymodel.RWIPosting) yacymodel.RWIPosting {
	return posting
}

func (p *Postings) Store(tx *vault.Txn, postings ...yacymodel.RWIPosting) error {
	for _, posting := range postings {
		if _, err := p.collection.Put(tx, identityOf(posting), posting); err != nil {
			return fmt.Errorf("store posting: %w", err)
		}
	}

	return nil
}

func identityOf(posting yacymodel.RWIPosting) postingIdentity {
	return postingIdentity{word: posting.WordHash, url: posting.URLHash}
}

func (p *Postings) Remove(tx *vault.Txn, postings ...yacymodel.RWIPosting) error {
	for _, posting := range postings {
		if _, err := p.collection.Delete(tx, identityOf(posting)); err != nil {
			return fmt.Errorf("remove posting: %w", err)
		}
	}

	return nil
}
