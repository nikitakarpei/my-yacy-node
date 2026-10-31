package rwipostings_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type storedProofCodec struct{}

func (storedProofCodec) Encode(proof yacymodel.PostingsProof) ([]byte, error) {
	return proof.Bytes(), nil
}

func (storedProofCodec) Decode(data []byte) (yacymodel.PostingsProof, error) {
	return yacymodel.ParsePostingsProofBytes(data)
}

var storedPostingKeys = vault.PairKey(hashkeypart.Hash, hashkeypart.URLHash).KeyLayoutFor(
	func(posting yacymodel.RWIPosting) (yacymodel.Hash, yacymodel.URLHash) {
		return posting.WordHash, posting.URLHash
	},
	func(word yacymodel.Hash, url yacymodel.URLHash) yacymodel.RWIPosting {
		return yacymodel.RWIPosting{WordHash: word, URLHash: url}
	},
)

func storeWithoutProjections(t *testing.T, engine vault.Engine, crawled yacymodel.PagePostings) {
	t.Helper()

	v, err := vault.New(engine, nil, nil, wallclock.Clock{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	postings, err := v.RegisterCollection("rwi", storedPostingKeys, rwipostings.PostingCodec())
	if err != nil {
		t.Fatalf("RegisterCollection: %v", err)
	}
	heldProofs, err := v.RegisterCollection(
		"rwi_held_proofs",
		vault.SingleKey(hashkeypart.URLHash).KeyLayout(),
		storedProofCodec{},
	)
	if err != nil {
		t.Fatalf("RegisterCollection: %v", err)
	}
	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		for _, posting := range crawled.Postings() {
			if _, err := postings.Put(tx, posting, posting); err != nil {
				return err
			}
		}
		proof, _ := crawled.Proof().Get()
		_, err := heldProofs.Put(tx, crawled.URL(), proof)

		return err
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func TestPostingsStoredBeforeTheURLPostingsIndexGetTheirProofs(t *testing.T) {
	engine := memoryvault.OpenEngine(0)
	crawled := pagePostingsFrom(t, "w1", "w2", "w3")
	storeWithoutProjections(t, engine, crawled)
	v, err := vault.New(engine, nil, nil, wallclock.Clock{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, _, _, proofs, err := rwipostings.Open(t.Context(), v, ringPartitions(t), nil, nil, nil)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}

	proof, found := proofs.ProofsFor(t.Context(), crawled.Postings()[1:])[crawled.URL()]
	if !found {
		t.Fatal("ProofsFor gave no proof for postings stored before the index")
	}
	proven, err := yacymodel.ProvenPagePostingsFrom(crawled.Postings()[1:], proof)
	if err != nil {
		t.Fatal(err)
	}
	if proven.Root() != crawled.Root() {
		t.Fatalf("root = %+v, want %+v", proven.Root(), crawled.Root())
	}
}
