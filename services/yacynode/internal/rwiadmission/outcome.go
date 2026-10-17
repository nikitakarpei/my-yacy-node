package rwiadmission

import (
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagefingerprint"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

type Outcome string

const (
	OutcomeHeldInEscrow           Outcome = "held_in_escrow"
	OutcomeIndexed                Outcome = "indexed"
	OutcomeDroppedAlreadyIndexed  Outcome = "dropped_already_indexed"
	OutcomeDroppedOtherVersion    Outcome = "dropped_other_version"
	OutcomeDroppedOtherNetwork    Outcome = "dropped_other_network"
	OutcomeDroppedFutureDate      Outcome = "dropped_future_date"
	OutcomeDroppedStopword        Outcome = "dropped_stopword"
	OutcomeRefusedTooManyPostings Outcome = "refused_too_many_postings"
	OutcomeRefusedStorageFull     Outcome = "refused_storage_full"
	OutcomeRefusedEscrowFull      Outcome = "refused_escrow_full"
)

type postingJudge struct {
	postings rwipostings.PostingIndex
	now      func() time.Time
}

func (j postingJudge) outcomeOf(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
	storedVersion yacymodel.Optional[urlmeta.CurrentVersion],
) (Outcome, error) {
	postingVersion := pagefingerprint.FromPosting(posting)
	currentVersion, metadataStored := storedVersion.Get()
	switch {
	case postingVersion.IsDatedAfter(j.now()):
		return OutcomeDroppedFutureDate, nil
	case !metadataStored || postingVersion.Modified > currentVersion.Fingerprint.Modified:
		return OutcomeHeldInEscrow, nil
	case postingVersion != currentVersion.Fingerprint:
		return OutcomeDroppedOtherVersion, nil
	case currentVersion.Source != urlmeta.SourcePeer(origin):
		return OutcomeDroppedOtherNetwork, nil
	}
	postingIndexed, err := j.isIndexed(tx, posting)
	if err != nil {
		return "", err
	}
	if postingIndexed {
		return OutcomeDroppedAlreadyIndexed, nil
	}

	return OutcomeIndexed, nil
}

func (j postingJudge) isIndexed(tx *vault.Txn, posting yacymodel.RWIPosting) (bool, error) {
	_, indexed, err := j.postings.PostingOf(tx, posting.WordHash, posting.URLHash)
	if err != nil {
		return false, fmt.Errorf("read indexed posting: %w", err)
	}

	return indexed, nil
}
