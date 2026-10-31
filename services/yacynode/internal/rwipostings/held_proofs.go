package rwipostings

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type heldProofs struct {
	proofs   *vault.Collection[yacymodel.URLHash, yacymodel.PostingsProof]
	urlWords *vault.MemberSetIndex[yacymodel.Hash, yacymodel.URLHash, postingIdentity]
	postings *vault.IndexedCollection[postingIdentity, yacymodel.RWIPosting]
}

func (h heldProofs) mergeWith(tx *vault.Txn, pagePostings yacymodel.PagePostings) error {
	pageProof, proven := pagePostings.Proof().Get()
	if !proven {
		return nil
	}
	url := pagePostings.URL()
	heldProof, found, err := h.proofOf(tx, url)
	if err != nil {
		return err
	}
	if found {
		return h.store(tx, url, heldProof.MergedWith(pageProof))
	}

	return h.store(tx, url, pageProof)
}

func (h heldProofs) proofOf(
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

func (h heldProofs) narrowWithout(tx *vault.Txn, purgedPosting yacymodel.RWIPosting) error {
	url := purgedPosting.URLHash
	heldProof, found, err := h.proofOf(tx, url)
	if err != nil || !found {
		return err
	}
	remainingWords, err := wordsOfURL(tx, h.urlWords, url)
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

func (h heldProofs) proofFor(
	ctx context.Context,
	tx *vault.Txn,
	offeredPostings []yacymodel.RWIPosting,
) (yacymodel.PostingsProof, bool, error) {
	url := offeredPostings[0].URLHash
	heldProof, found, err := h.proofOf(tx, url)
	if err != nil || !found {
		return yacymodel.PostingsProof{}, false, err
	}
	heldPostings, err := h.postingsOf(tx, url)
	if err != nil {
		return yacymodel.PostingsProof{}, false, err
	}
	held, err := yacymodel.ProvenPagePostingsFrom(heldPostings, heldProof)
	if err != nil {
		reportHeldProofBesideItsPostings(ctx, url, err)

		return yacymodel.PostingsProof{}, false, nil
	}
	given, stillHeld := held.NarrowedTo(offeredPostings).Get()
	if !stillHeld {
		reportOfferedPostingsNoLongerHeld(ctx, url)

		return yacymodel.PostingsProof{}, false, nil
	}
	proof, present := given.Proof().Get()

	return proof, present, nil
}

func (h heldProofs) postingsOf(
	tx *vault.Txn,
	url yacymodel.URLHash,
) ([]yacymodel.RWIPosting, error) {
	words, err := wordsOfURL(tx, h.urlWords, url)
	if err != nil {
		return nil, err
	}
	postings := make([]yacymodel.RWIPosting, 0, len(words))
	for _, word := range words {
		identity := postingIdentity{word: word, url: url}
		storedPosting, found, err := h.postings.Get(tx, identity)
		if err != nil {
			return nil, fmt.Errorf("read rwi posting: %w", err)
		}
		if found {
			postings = append(postings, postingWithIdentity(identity, storedPosting))
		}
	}

	return postings, nil
}

func reportHeldProofBesideItsPostings(ctx context.Context, url yacymodel.URLHash, err error) {
	slog.ErrorContext(ctx, "held postings of a url form no page postings",
		slog.String("url", url.String()),
		slog.Any("error", err),
	)
}

func reportOfferedPostingsNoLongerHeld(ctx context.Context, url yacymodel.URLHash) {
	slog.WarnContext(ctx, "offered postings of a url are no longer held",
		slog.String("url", url.String()),
	)
}
