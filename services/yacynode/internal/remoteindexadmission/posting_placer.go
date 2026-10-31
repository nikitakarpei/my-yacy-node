package remoteindexadmission

import (
	"context"
	"fmt"
	"log/slog"

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
	pagePosting yacymodel.PagePosting,
	arrivedVersion pageversion.Version,
	storedVersion pageversion.Version,
) (PostingOutcome, error) {
	switch storedVersion.JoiningOf(arrivedVersion) {
	case pageversion.JoiningRefusedOtherVersion:
		return PostingDroppedNotNewerVersion, nil
	case pageversion.JoiningRefusedOtherSource:
		return PostingDroppedPageFromOtherSource, nil
	case pageversion.JoiningRefusedUnrootedVersion:
		return PostingDroppedUnrootedVersion, nil
	case pageversion.JoiningAllowed:
	}
	postingIndexed, err := p.isIndexed(tx, pagePosting)
	if err != nil {
		return "", err
	}
	if postingIndexed {
		return PostingDroppedAlreadyIndexed, nil
	}

	return PostingIndexed, nil
}

func (p postingPlacer) isIndexed(tx *vault.Txn, pagePosting yacymodel.PagePosting) (bool, error) {
	_, indexed, err := p.postings.PostingOf(tx, pagePosting.Word(), pagePosting.URL())
	if err != nil {
		return false, fmt.Errorf("read indexed posting: %w", err)
	}

	return indexed, nil
}

func (p postingPlacer) indexJoined(
	ctx context.Context,
	tx *vault.Txn,
	passedPagePostings []yacymodel.PagePosting,
) error {
	if len(passedPagePostings) == 0 {
		return nil
	}
	joinedPagePostings, err := yacymodel.PagePostingsJoinedFrom(passedPagePostings)
	if err != nil {
		reportUnjoinedPassedPostings(ctx, passedPagePostings[0].URL(), err)

		return nil
	}
	if err := p.writer.Write(tx, joinedPagePostings); err != nil {
		return fmt.Errorf("write postings: %w", err)
	}

	return nil
}

func reportUnjoinedPassedPostings(ctx context.Context, url yacymodel.URLHash, err error) {
	slog.ErrorContext(ctx, "passed postings of a url form no page postings",
		slog.String("url", url.String()),
		slog.Any("error", err),
	)
}
