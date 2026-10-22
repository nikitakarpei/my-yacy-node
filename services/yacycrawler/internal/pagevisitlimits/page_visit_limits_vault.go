package pagevisitlimits

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
	takenHostPageCountsBucket = "taken_host_page_counts"
	usedAllowancesBucket      = "used_page_visit_allowances"
)

var errLimitMalformed = errors.New("page visit limit record is malformed")

func registerPageVisitLimits(
	ctx context.Context,
	storage *vault.Vault,
	keptFor time.Duration,
) (
	*vault.ExpiringCollection[hostKey, takenHostPageCount],
	*vault.ExpiringCollection[pageVisitKey, usedAllowances],
	error,
) {
	takenHostPageCounts, err := storage.RegisterExpiringCollection(
		ctx, takenHostPageCountsBucket, hostKeys, takenHostPageCountCodec{}, keptFor, nil,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("register %s: %w", takenHostPageCountsBucket, err)
	}
	usedAllowances, err := storage.RegisterExpiringCollection(
		ctx, usedAllowancesBucket, pageVisitKeys, usedAllowancesCodec{}, keptFor, nil,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("register %s: %w", usedAllowancesBucket, err)
	}
	return takenHostPageCounts, usedAllowances, nil
}

var hostKeys = vault.PairKey(vaultkeyparts.CrawlID, vault.TextKeyPart).KeyLayoutFor(
	func(key hostKey) (crawlid.CrawlID, string) { return key.crawlID, key.host },
	func(crawlID crawlid.CrawlID, host string) hostKey {
		return hostKey{crawlID: crawlID, host: host}
	},
)

var pageVisitKeys = vault.PairKey(vaultkeyparts.CrawlID, vaultkeyparts.URLHash).KeyLayoutFor(
	func(key pageVisitKey) (crawlid.CrawlID, urlhash.URLHash) { return key.crawlID, key.url },
	func(crawlID crawlid.CrawlID, url urlhash.URLHash) pageVisitKey {
		return pageVisitKey{crawlID: crawlID, url: url}
	},
)

type takenHostPageCountCodec struct{}

func (takenHostPageCountCodec) Encode(record takenHostPageCount) ([]byte, error) {
	return binary.AppendVarint(nil, int64(record.pages)), nil
}

func (takenHostPageCountCodec) Decode(stored []byte) (takenHostPageCount, error) {
	pages, pagesBytes := binary.Varint(stored)
	if pagesBytes != len(stored) {
		return takenHostPageCount{}, errLimitMalformed
	}
	return takenHostPageCount{pages: int(pages)}, nil
}

type usedAllowancesCodec struct{}

func (usedAllowancesCodec) Encode(record usedAllowances) ([]byte, error) {
	encoded := make([]byte, 0, 1+2*binary.MaxVarintLen64)
	if record.pageOfHost {
		encoded = append(encoded, 1)
	} else {
		encoded = append(encoded, 0)
	}
	encoded = binary.AppendVarint(encoded, int64(record.deferrals))
	return binary.AppendVarint(encoded, int64(record.attempts)), nil
}

func (usedAllowancesCodec) Decode(stored []byte) (usedAllowances, error) {
	if len(stored) == 0 {
		return usedAllowances{}, errLimitMalformed
	}
	deferrals, deferralsBytes := binary.Varint(stored[1:])
	if deferralsBytes <= 0 {
		return usedAllowances{}, errLimitMalformed
	}
	attempts, attemptsBytes := binary.Varint(stored[1+deferralsBytes:])
	if attemptsBytes <= 0 {
		return usedAllowances{}, errLimitMalformed
	}
	return usedAllowances{
		pageOfHost: stored[0] == 1,
		deferrals:  int(deferrals),
		attempts:   int(attempts),
	}, nil
}
