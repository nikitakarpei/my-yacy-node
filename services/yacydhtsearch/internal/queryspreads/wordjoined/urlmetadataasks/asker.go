// Package urlmetadataasks asks the peers that listed the joined documents for
// their URL metadata, and ends once the answers cover them, every ask
// settled, or the cutoff passed.
package urlmetadataasks

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/wordpartitionasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PeerAsks interface {
	AskForURLMetadata(
		ctx context.Context,
		asks []peerasks.URLMetadataAsk,
	) <-chan peerasks.URLMetadataAskOutcome
}

type Ceilings interface {
	CeilingOf(ctx context.Context, address string) int
}

type Clock interface {
	After(timeout time.Duration, expire func()) (stop func())
}

type Asker struct {
	peerAsks                    PeerAsks
	ceilings                    Ceilings
	cutoff                      Cutoff
	clock                       Clock
	amountOfPeersHoldingOneWord int
}

func New(
	peerAsks PeerAsks,
	ceilings Ceilings,
	cutoff Cutoff,
	clock Clock,
	amountOfPeersHoldingOneWord int,
) Asker {
	return Asker{
		peerAsks:                    peerAsks,
		ceilings:                    ceilings,
		cutoff:                      cutoff,
		clock:                       clock,
		amountOfPeersHoldingOneWord: amountOfPeersHoldingOneWord,
	}
}

func (asker Asker) AskFor(
	ctx context.Context,
	documentsMostHeldFirst []yacymodel.URLHash,
	answers []wordpartitionasks.ReplicaAnswer,
) Answers {
	asks := asker.asksFor(ctx, documentsMostHeldFirst, answers)
	asksContext, endAsks := context.WithCancel(ctx)
	defer endAsks()

	return runOf(asks).settleUntilEnded(
		asker.peerAsks.AskForURLMetadata(asksContext, asks),
		asker.cutoff,
		asker.clock,
	)
}

func (asker Asker) asksFor(
	ctx context.Context,
	documentsMostHeldFirst []yacymodel.URLHash,
	answers []wordpartitionasks.ReplicaAnswer,
) []peerasks.URLMetadataAsk {
	asksOfEachPeer := peersWithTheirAbstractsFrom(answers).asksFor(
		ctx,
		documentsMostHeldFirst,
		asker.ceilings,
	)

	return asksCoveringMostDocuments(asksOfEachPeer, asker.amountOfPeersHoldingOneWord)
}

func asksCoveringMostDocuments(
	asks []peerasks.URLMetadataAsk,
	amountOfPeersHoldingOneWord int,
) []peerasks.URLMetadataAsk {
	if len(asks) <= amountOfPeersHoldingOneWord {
		return asks
	}

	coveringAsks := make([]peerasks.URLMetadataAsk, 0, amountOfPeersHoldingOneWord)
	coveredDocuments := yacymodel.URLHashes{}
	for len(coveringAsks) < amountOfPeersHoldingOneWord {
		place, found := placeOfMostCoveringAskAmong(asks, coveredDocuments)
		if !found {
			break
		}
		coveringAsks = append(coveringAsks, asks[place])
		coveredDocuments.AddEach(asks[place].Documents)
	}

	return coveringAsks
}

func placeOfMostCoveringAskAmong(
	asks []peerasks.URLMetadataAsk,
	coveredDocuments yacymodel.URLHashes,
) (int, bool) {
	placeOfMostCoveringAsk := 0
	mostUncoveredDocuments := 0
	for place, ask := range asks {
		amountOfUncoveredDocuments := amountOfDocumentsNotCovered(ask.Documents, coveredDocuments)
		if amountOfUncoveredDocuments > mostUncoveredDocuments {
			placeOfMostCoveringAsk = place
			mostUncoveredDocuments = amountOfUncoveredDocuments
		}
	}

	return placeOfMostCoveringAsk, mostUncoveredDocuments > 0
}

func amountOfDocumentsNotCovered(
	documents []yacymodel.URLHash,
	coveredDocuments yacymodel.URLHashes,
) int {
	amount := 0
	for _, document := range documents {
		if coveredDocuments.Contains(document) {
			continue
		}
		amount++
	}

	return amount
}
