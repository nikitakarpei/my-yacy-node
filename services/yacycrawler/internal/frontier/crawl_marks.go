package frontier

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

const crawlMarksBucket = "crawl_marks"

var errCrawlMarkMalformed = errors.New("crawl mark has a malformed kind")

type crawlMarks struct {
	ofCrawl *vault.Collection[crawlid.CrawlID, crawlMark]
}

type crawlMark int64

const (
	unmarked crawlMark = iota
	markClosed
)

func (mark crawlMark) takesPageVisits() bool {
	return mark == unmarked
}

func registerCrawlMarks(storage *vault.Vault) (crawlMarks, error) {
	ofCrawl, err := storage.RegisterCollection(
		crawlMarksBucket, vault.SingleKey(vaultkeyparts.CrawlID).KeyLayout(), crawlMarkCodec{},
	)
	if err != nil {
		return crawlMarks{}, fmt.Errorf("register %s: %w", crawlMarksBucket, err)
	}
	return crawlMarks{ofCrawl: ofCrawl}, nil
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
