package rwipostings

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const heldProofsBucket vault.Name = "rwi_held_proofs"

func registerHeldProofs(
	v *vault.Vault,
) (*vault.Collection[yacymodel.URLHash, yacymodel.PostingsProof], error) {
	proofs, err := v.RegisterCollection(
		heldProofsBucket,
		vault.SingleKey(hashkeypart.URLHash).KeyLayout(),
		heldProofValueCodec{},
	)
	if err != nil {
		return nil, fmt.Errorf("register held proofs: %w", err)
	}

	return proofs, nil
}

type heldProofValueCodec struct{}

func (heldProofValueCodec) Encode(proof yacymodel.PostingsProof) ([]byte, error) {
	return proof.Bytes(), nil
}

func (heldProofValueCodec) Decode(data []byte) (yacymodel.PostingsProof, error) {
	return yacymodel.ParsePostingsProofBytes(data)
}
