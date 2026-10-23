package frontier

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

const (
	crawlHeadsBucket      = "crawl_heads"
	crawlHeadsByDueBucket = "crawl_heads_by_due"
)

var errCrawlHeadMalformed = errors.New("crawl head has a malformed due time")

type crawlHeads struct {
	dueOfCrawl *vault.Collection[crawlid.CrawlID, int64]
	byDue      *vault.ValueIndex[crawlHead, crawlHead]
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
	if heads.dueOfCrawl, err = storage.RegisterProjectedCollection(
		ctx, crawlHeadsBucket, vault.SingleKey(vaultkeyparts.CrawlID).KeyLayout(), headDueCodec{},
		vault.ProjectedAs(func(crawlID crawlid.CrawlID, due int64) crawlHead {
			return crawlHead{due: due, crawlID: crawlID}
		}, heads.byDue),
	); err != nil {
		return crawlHeads{}, fmt.Errorf("register %s: %w", crawlHeadsBucket, err)
	}
	return heads, nil
}

func (heads crawlHeads) moveEarlierTo(tx *vault.Txn, crawlID crawlid.CrawlID, due int64) error {
	headDue, found, err := heads.dueOfCrawl.Get(tx, crawlID)
	if err != nil || (found && headDue <= due) {
		return err
	}
	return heads.moveTo(tx, crawlID, due)
}

func (heads crawlHeads) moveTo(tx *vault.Txn, crawlID crawlid.CrawlID, due int64) error {
	_, err := heads.dueOfCrawl.Put(tx, crawlID, due)
	return err
}

func (heads crawlHeads) remove(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	_, err := heads.dueOfCrawl.Delete(tx, crawlID)
	return err
}

func (heads crawlHeads) firstDueBy(tx *vault.Txn, now time.Time) (crawlHead, bool, error) {
	return heads.byDue.First(tx, crawlHeadKeyParts.KeysThroughFirst(now.Unix()))
}

func (heads crawlHeads) has(tx *vault.Txn, crawlID crawlid.CrawlID) (bool, error) {
	_, found, err := heads.dueOfCrawl.Get(tx, crawlID)
	return found, err
}

var crawlHeadKeyParts = vault.PairKey(vault.IntegerKeyPart, vaultkeyparts.CrawlID)

var crawlHeadEntries = crawlHeadKeyParts.KeyLayoutFor(
	func(head crawlHead) (int64, crawlid.CrawlID) { return head.due, head.crawlID },
	func(due int64, crawlID crawlid.CrawlID) crawlHead {
		return crawlHead{due: due, crawlID: crawlID}
	},
)

type headDueCodec struct{}

func (headDueCodec) Encode(due int64) ([]byte, error) {
	return binary.AppendVarint(nil, due), nil
}

func (headDueCodec) Decode(stored []byte) (int64, error) {
	due, read := binary.Varint(stored)
	if read <= 0 {
		return 0, fmt.Errorf("%w: %d bytes", errCrawlHeadMalformed, len(stored))
	}
	return due, nil
}
