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
	OutcomeHeldInEscrow           Outcome = "held_in_escrow"
	OutcomeIndexedFirstVersion    Outcome = "indexed_first_version"
	OutcomeIndexedMissingWord     Outcome = "indexed_missing_word"
	OutcomeIndexedNewerVersion    Outcome = "indexed_newer_version"
	OutcomeDroppedSameVersion     Outcome = "dropped_same_version"
	OutcomeDroppedOtherVersion    Outcome = "dropped_other_version"
	OutcomeDroppedFutureDate      Outcome = "dropped_future_date"
	OutcomeDroppedOverSwapLimit   Outcome = "dropped_over_swap_limit"
	OutcomeDroppedStopword        Outcome = "dropped_stopword"
	OutcomeRefusedTooManyPostings Outcome = "refused_too_many_postings"
	OutcomeRefusedStorageFull     Outcome = "refused_storage_full"
	OutcomeRefusedEscrowFull      Outcome = "refused_escrow_full"
)

type postingJudge struct {
	indexedVersions IndexedVersions
	postings        rwipostings.PostingIndex
	swapAllowance   *swapAllowance
	now             func() time.Time
}

func (j postingJudge) outcomeOf(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
	metadataVersion yacymodel.Optional[pagefingerprint.Fingerprint],
) (Outcome, error) {
	postingVersion := pagefingerprint.FromPosting(posting)
	currentVersion, metadataStored := metadataVersion.Get()
	switch {
	case postingVersion.IsDatedAfter(j.now()):
		return OutcomeDroppedFutureDate, nil
	case !metadataStored || postingVersion.Modified > currentVersion.Modified:
		return OutcomeHeldInEscrow, nil
	case postingVersion != currentVersion:
		return OutcomeDroppedOtherVersion, nil
	}

	return j.outcomeAgainstIndexedVersion(tx, origin, posting, postingVersion)
}

func (j postingJudge) outcomeAgainstIndexedVersion(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
	postingVersion pagefingerprint.Fingerprint,
) (Outcome, error) {
	indexedVersion, versionIndexed, err := j.indexedVersions.Of(tx, posting.URLHash)
	if err != nil {
		return "", err
	}
	if !versionIndexed {
		return OutcomeIndexedFirstVersion, nil
	}
	if indexedVersion.Fingerprint != postingVersion {
		allowed, err := j.swapAllowance.allows(tx, origin.Network)
		if err != nil {
			return "", err
		}
		if !allowed {
			return OutcomeDroppedOverSwapLimit, nil
		}

		return OutcomeIndexedNewerVersion, nil
	}
	if indexedVersion.Origin.Network != origin.Network {
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
