package undeliveredpostings

import (
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/hashkeypart"
)

const undeliveredPostingsBucket vault.Name = "rwidistribution_undelivered_postings"

const undeliveredPostingsVersion = "1"

func registerUndeliveredPostings(
	v *vault.Vault,
	partitions yacymodel.DHTRingPartitions,
) (*vault.ChangeIndex[yacymodel.RWIPosting, undeliveredEntry], error) {
	postings, err := v.RegisterChangeIndex(
		undeliveredPostingsBucket,
		undeliveredPostingsVersion,
		undeliveredEntryKeyLayout,
		func(posting yacymodel.RWIPosting) undeliveredEntry {
			return undeliveredEntryOf(posting, partitions)
		},
	)
	if err != nil {
		return nil, fmt.Errorf("register undelivered postings: %w", err)
	}

	return postings, nil
}

type undeliveredEntry struct {
	position yacymodel.DHTRingPosition
	word     yacymodel.Hash
	url      yacymodel.URLHash
}

func undeliveredEntryOf(
	posting yacymodel.RWIPosting,
	partitions yacymodel.DHTRingPartitions,
) undeliveredEntry {
	return undeliveredEntry{
		position: yacymodel.DHTRingPositionOfPosting(posting, partitions),
		word:     posting.WordHash,
		url:      posting.URLHash,
	}
}

var undeliveredEntryKeyParts = vault.TripleKey(
	vault.IntegerKeyPart,
	hashkeypart.Hash,
	hashkeypart.URLHash,
)

var undeliveredEntryKeyLayout = undeliveredEntryKeyParts.KeyLayoutFor(
	func(entry undeliveredEntry) (int64, yacymodel.Hash, yacymodel.URLHash) {
		return positionKeyOf(entry.position), entry.word, entry.url
	},
	func(position int64, word yacymodel.Hash, url yacymodel.URLHash) undeliveredEntry {
		return undeliveredEntry{position: positionFromKey(position), word: word, url: url}
	},
)

func positionKeyOf(position yacymodel.DHTRingPosition) int64 {
	return int64(position) //nolint:gosec // G115: ring positions stay below 1<<63.
}

func positionFromKey(positionKey int64) yacymodel.DHTRingPosition {
	//nolint:gosec // G115: keys hold ring positions only.
	return yacymodel.DHTRingPosition(positionKey)
}

func entriesIn(sector yacymodel.DHTRingSector, lastVisited UndeliveredPosting) vault.KeyRange {
	highestPosition := positionKeyOf(yacymodel.HighestDHTRingPositionIn(sector))
	if lastVisited == (UndeliveredPosting{}) {
		return undeliveredEntryKeyParts.KeysBetweenFirsts(
			positionKeyOf(yacymodel.LowestDHTRingPositionIn(sector)),
			highestPosition,
		)
	}

	return undeliveredEntryKeyParts.KeysAfterKeyThroughFirst(
		positionKeyOf(lastVisited.change.Entry.position),
		lastVisited.change.Entry.word,
		lastVisited.change.Entry.url,
		highestPosition,
	)
}

func entriesOf(
	wordPartition yacymodel.WordPartition,
	partitions yacymodel.DHTRingPartitions,
) vault.KeyRange {
	return undeliveredEntryKeyParts.KeysWithFirstAndSecond(
		positionKeyOf(yacymodel.DHTRingPositionOfWordPartition(wordPartition, partitions)),
		wordPartition.Word,
	)
}
