package rwiadmission

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type Outcome string

const (
	OutcomeHeldInEscrow         Outcome = "held_in_escrow"
	OutcomeIndexedFirstVersion  Outcome = "indexed_first_version"
	OutcomeIndexedMissingWord   Outcome = "indexed_missing_word"
	OutcomeIndexedNewerVersion  Outcome = "indexed_newer_version"
	OutcomeDroppedSameVersion   Outcome = "dropped_same_version"
	OutcomeDroppedOtherVersion  Outcome = "dropped_other_version"
	OutcomeDroppedFutureDate    Outcome = "dropped_future_date"
	OutcomeDroppedOverSwapLimit Outcome = "dropped_over_swap_limit"
)

type postingJudge struct {
	indexedVersions IndexedVersions
	postings        rwipostings.PostingIndex
	swaps           *swapAllowance
	now             func() time.Time
}

func (j postingJudge) outcomeOf(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
	storedFingerprint yacymodel.Optional[pagefingerprint.Fingerprint],
) (Outcome, error) {
	postingFingerprint := pagefingerprint.OfPosting(posting)
	metadataFingerprint, metadataStored := storedFingerprint.Get()
	switch {
	case postingFingerprint.IsDatedAfter(j.now()):
		return OutcomeDroppedFutureDate, nil
	case !metadataStored || postingFingerprint.Modified > metadataFingerprint.Modified:
		return OutcomeHeldInEscrow, nil
	case postingFingerprint != metadataFingerprint:
		return OutcomeDroppedOtherVersion, nil
	}

	return j.outcomeAgainstIndexedVersion(tx, origin, posting, postingFingerprint)
}

func (j postingJudge) outcomeAgainstIndexedVersion(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
	postingFingerprint pagefingerprint.Fingerprint,
) (Outcome, error) {
	indexedVersion, versionIndexed, err := j.indexedVersions.Of(tx, posting.URLHash)
	if err != nil {
		return "", err
	}
	if !versionIndexed {
		return OutcomeIndexedFirstVersion, nil
	}
	if indexedVersion.Fingerprint != postingFingerprint {
		allowed, err := j.swaps.allows(tx, origin.Network)
		if err != nil {
			return "", err
		}
		if !allowed {
			return OutcomeDroppedOverSwapLimit, nil
		}

		return OutcomeIndexedNewerVersion, nil
	}
	if indexedVersion.Origin != origin {
		return OutcomeDroppedSameVersion, nil
	}
	postingIndexed, err := j.isIndexed(tx, posting)
	if err != nil {
		return "", err
	}
	if postingIndexed {
		return OutcomeDroppedSameVersion, nil
	}

	return OutcomeIndexedMissingWord, nil
}

func (j postingJudge) isIndexed(tx *vault.Txn, posting yacymodel.RWIPosting) (bool, error) {
	_, indexed, err := j.postings.PostingOf(tx, posting.WordHash, posting.URLHash)
	if err != nil {
		return false, fmt.Errorf("read indexed posting: %w", err)
	}

	return indexed, nil
}
