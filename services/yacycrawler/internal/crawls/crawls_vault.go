package crawls

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/vaultkeyparts"
)

const (
	unfinishedCrawlsBucket                 = "unfinished_crawls"
	finishedCrawlsBucket                   = "finished_crawls"
	amountsOfUnfinishedCrawlsInStateBucket = "amounts_of_unfinished_crawls_in_state"
	crawlIndexesVersion                    = "1"
)

func registerCrawls(
	ctx context.Context,
	storage *vault.Vault,
	retention time.Duration,
	forgotten ForgottenCrawls,
) (*Crawls, error) {
	c := &Crawls{}
	var err error
	if c.unfinishedIndexes, err = registerCrawlIndexes(
		storage,
		unfinishedCrawlsBucket,
	); err != nil {
		return nil, err
	}
	if c.finishedIndexes, err = registerCrawlIndexes(storage, finishedCrawlsBucket); err != nil {
		return nil, err
	}
	if c.amountsInState, err = storage.RegisterCountIndex(
		amountsOfUnfinishedCrawlsInStateBucket, crawlIndexesVersion,
		vault.SingleKey(stateKeyPart).KeyLayout(),
		func(crawl crawlWithID) yacycrawlcontract.CrawlState { return crawl.crawl.State },
	); err != nil {
		return nil, fmt.Errorf("register %s: %w", amountsOfUnfinishedCrawlsInStateBucket, err)
	}
	if c.unfinished, err = storage.RegisterProjectedCollection(
		ctx, unfinishedCrawlsBucket, crawlIDKeys, crawlCodec{},
		c.unfinishedIndexes.projectedWith(c.amountsInState),
	); err != nil {
		return nil, fmt.Errorf("register %s: %w", unfinishedCrawlsBucket, err)
	}
	if c.finished, err = storage.RegisterExpiringCollection(
		ctx, finishedCrawlsBucket, crawlIDKeys, crawlCodec{}, retention,
		c.finishedIndexes.projectedWith(), forgotten.CrawlForgotten,
	); err != nil {
		return nil, fmt.Errorf("register %s: %w", finishedCrawlsBucket, err)
	}
	return c, nil
}

var crawlIDKeys = vault.SingleKey(vaultkeyparts.CrawlID).KeyLayout()

type crawlWithID struct {
	crawlID crawlid.CrawlID
	crawl   Crawl
}

var stateKeyPart = vault.TextKeyPartFrom(
	func(state yacycrawlcontract.CrawlState) string { return string(state) },
	func(text string) (yacycrawlcontract.CrawlState, error) {
		return yacycrawlcontract.CrawlState(text), nil
	},
)

type crawlCodec struct{}

type storedCrawl struct {
	Order      yacycrawlcontract.CrawlOrder `json:"Order"`
	State      yacycrawlcontract.CrawlState `json:"State"`
	StartedAt  time.Time                    `json:"StartedAt"`
	FinishedAt time.Time                    `json:"FinishedAt,omitzero"`
}

func (crawlCodec) Encode(crawl Crawl) ([]byte, error) {
	encoded, err := json.Marshal(storedCrawl(crawl))
	if err != nil {
		return nil, fmt.Errorf("write the crawl: %w", err)
	}
	return encoded, nil
}

func (crawlCodec) Decode(stored []byte) (Crawl, error) {
	var crawl storedCrawl
	if err := json.Unmarshal(stored, &crawl); err != nil {
		return Crawl{}, fmt.Errorf("read the crawl: %w", err)
	}
	return Crawl(crawl), nil
}
