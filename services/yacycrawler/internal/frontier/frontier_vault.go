package frontier

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

const (
	frontierBucket      = "frontier"
	frontierByDueBucket = "frontier_by_due"
	dueTimesVersion     = "1"
	messageIDBytes      = 16
)

var (
	errPendingEntryTruncated = errors.New("pending page visit entry shorter than its header")
	errPendingEntryMalformed = errors.New("pending page visit entry has a malformed time")
)

func registerFrontier(
	ctx context.Context,
	storage *vault.Vault,
) (
	*vault.Collection[pageVisitKey, pendingEntry],
	*vault.ValueIndex[dueEntry, dueEntry],
	error,
) {
	dueTimes, err := storage.RegisterValueIndex(
		frontierByDueBucket,
		dueTimesVersion,
		dueEntries,
		func(entry dueEntry) dueEntry { return entry },
	)
	if err != nil {
		return nil, nil, fmt.Errorf("register %s: %w", frontierByDueBucket, err)
	}
	pageVisits, err := storage.RegisterProjectedCollection(
		ctx,
		frontierBucket,
		pageVisitKeys,
		pendingEntryCodec{},
		vault.ProjectedAs(func(key pageVisitKey, entry pendingEntry) dueEntry {
			return dueEntry{due: entry.due, key: key}
		}, dueTimes),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("register %s: %w", frontierBucket, err)
	}
	return pageVisits, dueTimes, nil
}

var pageVisitKeys = vault.PairKey(vaultkeyparts.CrawlID, vaultkeyparts.URLHash).KeyLayoutFor(
	func(key pageVisitKey) (crawlid.CrawlID, urlhash.URLHash) { return key.crawlID, key.url },
	func(crawlID crawlid.CrawlID, url urlhash.URLHash) pageVisitKey {
		return pageVisitKey{crawlID: crawlID, url: url}
	},
)

var dueKeyParts = vault.TripleKey(
	vault.IntegerKeyPart,
	vaultkeyparts.CrawlID,
	vaultkeyparts.URLHash,
)

var dueEntries = dueKeyParts.KeyLayoutFor(
	func(entry dueEntry) (int64, crawlid.CrawlID, urlhash.URLHash) {
		return entry.due, entry.key.crawlID, entry.key.url
	},
	func(due int64, crawlID crawlid.CrawlID, url urlhash.URLHash) dueEntry {
		return dueEntry{due: due, key: pageVisitKey{crawlID: crawlID, url: url}}
	},
)

type pendingEntryCodec struct{}

func (pendingEntryCodec) Encode(entry pendingEntry) ([]byte, error) {
	encoded := make([]byte, 0, 2*binary.MaxVarintLen64+messageIDBytes+len(entry.body))
	encoded = binary.AppendVarint(encoded, entry.due)
	encoded = binary.AppendVarint(encoded, entry.arrival)
	encoded = append(encoded, entry.messageID[:]...)
	return append(encoded, entry.body...), nil
}

func (pendingEntryCodec) Decode(stored []byte) (pendingEntry, error) {
	due, dueBytes := binary.Varint(stored)
	if dueBytes <= 0 {
		return pendingEntry{}, errPendingEntryMalformed
	}
	arrival, arrivalBytes := binary.Varint(stored[dueBytes:])
	if arrivalBytes <= 0 {
		return pendingEntry{}, errPendingEntryMalformed
	}
	rest := stored[dueBytes+arrivalBytes:]
	if len(rest) < messageIDBytes {
		return pendingEntry{}, fmt.Errorf("%w: %d bytes", errPendingEntryTruncated, len(stored))
	}
	entry := pendingEntry{
		due:     due,
		arrival: arrival,
		body:    append([]byte(nil), rest[messageIDBytes:]...),
	}
	copy(entry.messageID[:], rest[:messageIDBytes])
	return entry, nil
}
