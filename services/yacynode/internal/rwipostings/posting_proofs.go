package rwipostings

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type postingProofs struct {
	vault      *vault.Vault
	heldProofs heldProofs
}

func (p postingProofs) ProofsFor(
	ctx context.Context,
	postings []yacymodel.RWIPosting,
) map[yacymodel.URLHash]yacymodel.PostingsProof {
	offeredPerURL, _ := yacymodel.PostingsPerURLFrom(postings)
	proofs := make(map[yacymodel.URLHash]yacymodel.PostingsProof)
	err := p.vault.View(ctx, func(tx *vault.Txn) error {
		for _, offered := range offeredPerURL {
			proof, present, err := p.heldProofs.proofFor(ctx, tx, offered)
			if err != nil {
				return err
			}
			if present {
				proofs[offered[0].URLHash] = proof
			}
		}

		return nil
	})
	if err != nil {
		slog.WarnContext(ctx, "postings proofs not read", slog.Any("error", err))

		return nil
	}

	return proofs
}
