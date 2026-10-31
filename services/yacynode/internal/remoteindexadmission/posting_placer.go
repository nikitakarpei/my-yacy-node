package remoteindexadmission

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
)

type postingPlacer struct {
	postings rwipostings.PostingIndex
	writer   rwipostings.PostingWriter
}

func (p postingPlacer) placementOf(
	tx *vault.Txn,
	posting yacymodel.RWIPosting,
	arrivedVersion pageversion.Version,
	storedVersion pageversion.Version,
) (PostingOutcome, error) {
	switch storedVersion.JoiningOf(arrivedVersion) {
	case pageversion.JoiningRefusedOtherVersion:
		return PostingDroppedNotNewerVersion, nil
	case pageversion.JoiningRefusedOtherSource:
		return PostingDroppedPageFromOtherSource, nil
	case pageversion.JoiningAllowed:
	}
	postingIndexed, err := p.isIndexed(tx, posting)
	if err != nil {
		return "", err
	}
	if postingIndexed {
		return PostingDroppedAlreadyIndexed, nil
	}

	return PostingIndexed, nil
}

func (p postingPlacer) isIndexed(tx *vault.Txn, posting yacymodel.RWIPosting) (bool, error) {
	_, indexed, err := p.postings.PostingOf(tx, posting.WordHash, posting.URLHash)
	if err != nil {
		return false, fmt.Errorf("read indexed posting: %w", err)
	}

	return indexed, nil
}

func (p postingPlacer) index(tx *vault.Txn, pagePostingsToIndex yacymodel.PagePostings) error {
	if err := p.writer.Write(tx, pagePostingsToIndex); err != nil {
		return fmt.Errorf("write postings: %w", err)
	}

	return nil
}
