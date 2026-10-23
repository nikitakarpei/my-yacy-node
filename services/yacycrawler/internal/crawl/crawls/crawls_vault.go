package crawls

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/vaultkeyparts"
)

const crawlIndexesVersion = "1"

func registerCrawls(
	ctx context.Context,
	storage *vault.Vault,
	retention time.Duration,
	forgotten ForgottenCrawls,
) (*Crawls, error) {
	unfinished, err := registerUnfinishedCrawls(ctx, storage)
	if err != nil {
		return nil, err
	}
	finished, err := registerFinishedCrawls(ctx, storage, retention, forgotten)
	if err != nil {
		return nil, err
	}
	return &Crawls{unfinished: unfinished, finished: finished}, nil
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
