package remoteindexadmission

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

type PostingOutcome string

const (
	PostingHeldInEscrow           PostingOutcome = "held_in_escrow"
	PostingIndexed                PostingOutcome = "indexed"
	PostingDroppedAlreadyIndexed  PostingOutcome = "dropped_already_indexed"
	PostingDroppedOtherVersion    PostingOutcome = "dropped_other_version"
	PostingDroppedOtherNetwork    PostingOutcome = "dropped_other_network"
	PostingDroppedFutureDate      PostingOutcome = "dropped_future_date"
	PostingDroppedStopword        PostingOutcome = "dropped_stopword"
	PostingRefusedTooManyPostings PostingOutcome = "refused_too_many_postings"
	PostingRefusedStorageFull     PostingOutcome = "refused_storage_full"
	PostingRefusedEscrowFull      PostingOutcome = "refused_escrow_full"
)

type postingJudge struct {
	postings rwipostings.PostingIndex
	now      func() time.Time
}

func (j postingJudge) outcomeFor(
	tx *vault.Txn,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
	storedVersion yacymodel.Optional[urlmeta.CurrentVersion],
) (PostingOutcome, error) {
	postingFingerprint := pagefingerprint.FromPosting(posting)
	currentVersion, metadataStored := storedVersion.Get()
	switch {
	case postingFingerprint.IsDatedAfter(j.now()):
		return PostingDroppedFutureDate, nil
	case !metadataStored || postingFingerprint.Modified > currentVersion.Fingerprint.Modified:
		return PostingHeldInEscrow, nil
	case postingFingerprint != currentVersion.Fingerprint:
		return PostingDroppedOtherVersion, nil
	case currentVersion.Source != urlmeta.SourcePeer(origin):
		return PostingDroppedOtherNetwork, nil
	}
	postingIndexed, err := j.isIndexed(tx, posting)
	if err != nil {
		return "", err
	}
	if postingIndexed {
		return PostingDroppedAlreadyIndexed, nil
	}

	return PostingIndexed, nil
}

func (j postingJudge) isIndexed(tx *vault.Txn, posting yacymodel.RWIPosting) (bool, error) {
	_, indexed, err := j.postings.PostingOf(tx, posting.WordHash, posting.URLHash)
	if err != nil {
		return false, fmt.Errorf("read indexed posting: %w", err)
	}

	return indexed, nil
}
