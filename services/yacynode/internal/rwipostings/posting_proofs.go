package rwipostings

import (
	"context"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type postingProofs struct {
	vault     *vault.Vault
	directory postingDirectory
}

func (p postingProofs) ProofsFor(
	ctx context.Context,
	postings []yacymodel.RWIPosting,
) map[yacymodel.URLHash]yacymodel.PostingsProof {
	proofs := make(map[yacymodel.URLHash]yacymodel.PostingsProof)
	err := p.vault.View(ctx, func(tx *vault.Txn) error {
		for url, postingsOfURL := range postingsPerURLOf(postings) {
			proof, present, err := p.directory.proofFor(ctx, tx, url, postingsOfURL)
			if err != nil {
				return err
			}
			if present {
				proofs[url] = proof
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

func postingsPerURLOf(
	postings []yacymodel.RWIPosting,
) map[yacymodel.URLHash][]yacymodel.RWIPosting {
	postingsPerURL := make(map[yacymodel.URLHash][]yacymodel.RWIPosting)
	for _, posting := range postings {
		postingsPerURL[posting.URLHash] = append(postingsPerURL[posting.URLHash], posting)
	}

	return postingsPerURL
}

var _ PostingProofs = postingProofs{}
