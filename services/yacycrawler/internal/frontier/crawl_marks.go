package frontier

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

const (
	crawlMarksBucket       = "crawl_marks"
	crawlMarksByMarkBucket = "crawl_marks_by_mark"
)

var errCrawlMarkMalformed = errors.New("crawl mark is malformed")

type crawlMarks struct {
	ofCrawl *vault.Collection[crawlid.CrawlID, crawlMark]
	byMark  *vault.ValueIndex[markedCrawl, markedCrawl]
}

type crawlMark int64

const (
	unmarked crawlMark = iota
	markHeld
	markCancelling
	markClosed
)

func (mark crawlMark) takesPageVisits() bool {
	return mark == unmarked || mark == markHeld
}

func registerCrawlMarks(ctx context.Context, storage *vault.Vault) (crawlMarks, error) {
	var marks crawlMarks
	var err error
	if marks.byMark, err = storage.RegisterValueIndex(
		crawlMarksByMarkBucket, frontierIndexesVersion, markedCrawlEntries,
		func(marked markedCrawl) markedCrawl { return marked },
	); err != nil {
		return crawlMarks{}, fmt.Errorf("register %s: %w", crawlMarksByMarkBucket, err)
	}
	if marks.ofCrawl, err = storage.RegisterProjectedCollection(
		ctx, crawlMarksBucket, vault.SingleKey(vaultkeyparts.CrawlID).KeyLayout(), crawlMarkCodec{},
		vault.ProjectedAs(func(crawlID crawlid.CrawlID, mark crawlMark) markedCrawl {
			return markedCrawl{mark: int64(mark), crawlID: crawlID}
		}, marks.byMark),
	); err != nil {
		return crawlMarks{}, fmt.Errorf("register %s: %w", crawlMarksBucket, err)
	}
	return marks, nil
}

func (marks crawlMarks) markOf(tx *vault.Txn, crawlID crawlid.CrawlID) (crawlMark, error) {
	mark, _, err := marks.ofCrawl.Get(tx, crawlID)
	return mark, err
}

func (marks crawlMarks) mark(tx *vault.Txn, crawlID crawlid.CrawlID, mark crawlMark) error {
	_, err := marks.ofCrawl.Put(tx, crawlID, mark)
	return err
}

func (marks crawlMarks) unmark(tx *vault.Txn, crawlID crawlid.CrawlID) error {
	_, err := marks.ofCrawl.Delete(tx, crawlID)
	return err
}

func (marks crawlMarks) firstCancellingCrawls(
	tx *vault.Txn,
	amount int,
) ([]crawlid.CrawlID, error) {
	var cancelling []crawlid.CrawlID
	err := marks.byMark.Scan(
		tx,
		markedCrawlKeyParts.KeysWithFirst(int64(markCancelling)),
		func(marked markedCrawl) (bool, error) {
			cancelling = append(cancelling, marked.crawlID)
			return len(cancelling) < amount, nil
		},
	)
	return cancelling, err
}

type markedCrawl struct {
	mark    int64
	crawlID crawlid.CrawlID
}

var markedCrawlKeyParts = vault.PairKey(vault.IntegerKeyPart, vaultkeyparts.CrawlID)

var markedCrawlEntries = markedCrawlKeyParts.KeyLayoutFor(
	func(marked markedCrawl) (int64, crawlid.CrawlID) { return marked.mark, marked.crawlID },
	func(mark int64, crawlID crawlid.CrawlID) markedCrawl {
		return markedCrawl{mark: mark, crawlID: crawlID}
	},
)

type crawlMarkCodec struct{}

func (crawlMarkCodec) Encode(mark crawlMark) ([]byte, error) {
	return binary.AppendVarint(nil, int64(mark)), nil
}

func (crawlMarkCodec) Decode(stored []byte) (crawlMark, error) {
	mark, read := binary.Varint(stored)
	if read <= 0 {
		return unmarked, fmt.Errorf("%w: %d bytes", errCrawlMarkMalformed, len(stored))
	}
	return crawlMark(mark), nil
}
