package rwipostings

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type heldProofs struct {
	proofs *vault.Collection[yacymodel.URLHash, yacymodel.PostingsProof]
}

func (h heldProofs) mergeWith(tx *vault.Txn, pagePostings yacymodel.PagePostings) error {
	pageProof, proven := pagePostings.Proof().Get()
	if !proven {
		return nil
	}
	url := pagePostings.URL()
	heldProof, found, err := h.heldProofOf(tx, url)
	if err != nil {
		return err
	}
	if found {
		return h.store(tx, url, heldProof.MergedWith(pageProof))
	}

	return h.store(tx, url, pageProof)
}

func (h heldProofs) heldProofOf(
	tx *vault.Txn,
	url yacymodel.URLHash,
) (yacymodel.PostingsProof, bool, error) {
	heldProof, found, err := h.proofs.Get(tx, url)
	if err != nil {
		return yacymodel.PostingsProof{}, false, fmt.Errorf("read held proof: %w", err)
	}

	return heldProof, found, nil
}

func (h heldProofs) store(
	tx *vault.Txn,
	url yacymodel.URLHash,
	heldProof yacymodel.PostingsProof,
) error {
	if _, err := h.proofs.Put(tx, url, heldProof); err != nil {
		return fmt.Errorf("store held proof: %w", err)
	}

	return nil
}

func (h heldProofs) narrowTo(
	tx *vault.Txn,
	remainingWordsOf func(*vault.Txn, yacymodel.URLHash) ([]yacymodel.Hash, error),
	purgedPosting yacymodel.RWIPosting,
) error {
	url := purgedPosting.URLHash
	heldProof, found, err := h.heldProofOf(tx, url)
	if err != nil || !found {
		return err
	}
	remainingWords, err := remainingWordsOf(tx, url)
	if err != nil {
		return err
	}
	if len(remainingWords) == 0 {
		return h.delete(tx, url)
	}
	purgedPostings := []yacymodel.RWIPosting{purgedPosting}

	return h.store(tx, url, heldProof.NarrowedTo(remainingWords, purgedPostings))
}

func (h heldProofs) delete(tx *vault.Txn, url yacymodel.URLHash) error {
	if _, err := h.proofs.Delete(tx, url); err != nil {
		return fmt.Errorf("delete held proof: %w", err)
	}

	return nil
}
