package frontier

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/urlhash"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

const (
	crawlHeadsBucket      = "crawl_heads"
	crawlHeadsByDueBucket = "crawl_heads_by_due"
)

var errCrawlHeadMalformed = errors.New("crawl head has a malformed due time")

type crawlHeads struct {
	positionOfCrawl *vault.Collection[crawlid.CrawlID, duePosition]
	byDue           *vault.ValueIndex[crawlHead, crawlHead]
}

type crawlHead struct {
	due     int64
	crawlID crawlid.CrawlID
}

func registerCrawlHeads(ctx context.Context, storage *vault.Vault) (crawlHeads, error) {
	var heads crawlHeads
	var err error
	if heads.byDue, err = storage.RegisterValueIndex(
		crawlHeadsByDueBucket, frontierIndexesVersion, crawlHeadEntries,
		func(head crawlHead) crawlHead { return head },
	); err != nil {
		return crawlHeads{}, fmt.Errorf("register %s: %w", crawlHeadsByDueBucket, err)
	}
	if heads.positionOfCrawl, err = storage.RegisterProjectedCollection(
		ctx,
		crawlHeadsBucket,
		vault.SingleKey(vaultkeyparts.CrawlID).KeyLayout(),
		headPositionCodec{},
		vault.ProjectedAs(func(crawlID crawlid.CrawlID, position duePosition) crawlHead {
			return crawlHead{due: position.due, crawlID: crawlID}
		}, heads.byDue),
	); err != nil {
		return crawlHeads{}, fmt.Errorf("register %s: %w", crawlHeadsBucket, err)
	}
	return heads, nil
}

func (heads crawlHeads) moveEarlierTo(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	position duePosition,
) error {
	headPosition, found, err := heads.positionOfCrawl.Get(tx, crawlID)
	if err != nil || (found && !position.isBefore(headPosition)) {
		return err
	}
	return heads.moveTo(tx, crawlID, position)
}

func (heads crawlHeads) moveTo(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
	position duePosition,
) error {
	_, err := heads.positionOfCrawl.Put(tx, crawlID, position)
	return err
}

func (heads crawlHeads) remove(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	_, err := heads.positionOfCrawl.Delete(tx, crawlID)
	return err
}

func (heads crawlHeads) firstDueBy(tx *vault.Txn, now time.Time) (crawlHead, bool, error) {
	return heads.byDue.First(tx, crawlHeadKeyParts.KeysThroughFirst(now.Unix()))
}

func (heads crawlHeads) positionOf(
	tx *vault.Txn,
	crawlID crawlid.CrawlID,
) (duePosition, bool, error) {
	return heads.positionOfCrawl.Get(tx, crawlID)
}

var crawlHeadKeyParts = vault.PairKey(vault.IntegerKeyPart, vaultkeyparts.CrawlID)

var crawlHeadEntries = crawlHeadKeyParts.KeyLayoutFor(
	func(head crawlHead) (int64, crawlid.CrawlID) { return head.due, head.crawlID },
	func(due int64, crawlID crawlid.CrawlID) crawlHead {
		return crawlHead{due: due, crawlID: crawlID}
	},
)

type headPositionCodec struct{}

func (headPositionCodec) Encode(position duePosition) ([]byte, error) {
	return append(binary.AppendVarint(nil, position.due), position.url[:]...), nil
}

func (headPositionCodec) Decode(stored []byte) (duePosition, error) {
	due, read := binary.Varint(stored)
	if read <= 0 || len(stored)-read != len(urlhash.URLHash{}) {
		return duePosition{}, fmt.Errorf("%w: %d bytes", errCrawlHeadMalformed, len(stored))
	}
	return duePosition{due: due, url: urlhash.URLHash(stored[read:])}, nil
}
