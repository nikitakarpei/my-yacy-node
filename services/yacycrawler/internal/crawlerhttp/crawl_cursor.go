package crawlerhttp

import (
	"encoding/base64"
	"errors"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
)

const (
	olderCursorWord = "older"
	newerCursorWord = "newer"
	cursorBoundary  = "|"
	cursorParts     = 2
)

var errUnreadableCursor = errors.New("unreadable cursor")

type crawlCursor interface {
	pageIn(
		tx *vault.Txn,
		stores crawlStores,
		states []yacycrawlcontract.CrawlState,
		limit int,
	) (yacycrawlcontract.CrawlPage, error)
}

func crawlCursorOf(cursor string) (crawlCursor, error) {
	if cursor == "" {
		return olderCrawlsCursor{}, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, errUnreadableCursor
	}
	parts := strings.SplitN(string(decoded), cursorBoundary, cursorParts)
	if len(parts) != cursorParts {
		return nil, errUnreadableCursor
	}
	var border crawlid.CrawlID
	if err := border.UnmarshalText([]byte(parts[1])); err != nil {
		return nil, errUnreadableCursor
	}
	switch parts[0] {
	case olderCursorWord:
		return olderCrawlsCursor{border: border}, nil
	case newerCursorWord:
		return newerCrawlsCursor{border: border}, nil
	default:
		return nil, errUnreadableCursor
	}
}

type olderCrawlsCursor struct {
	border crawlid.CrawlID
}

func (cursor olderCrawlsCursor) pageIn(
	tx *vault.Txn,
	stores crawlStores,
	states []yacycrawlcontract.CrawlState,
	limit int,
) (yacycrawlcontract.CrawlPage, error) {
	crawls, err := stores.crawlsOlderThan(tx, states, cursor.border, limit+1)
	if err != nil {
		return yacycrawlcontract.CrawlPage{}, err
	}
	shown := crawls[:min(len(crawls), limit)]
	return crawlPageOf(shown, len(crawls) > limit, cursor.border != crawlid.CrawlID{}), nil
}

type newerCrawlsCursor struct {
	border crawlid.CrawlID
}

func (cursor newerCrawlsCursor) pageIn(
	tx *vault.Txn,
	stores crawlStores,
	states []yacycrawlcontract.CrawlState,
	limit int,
) (yacycrawlcontract.CrawlPage, error) {
	crawls, err := stores.crawlsNewerThan(tx, states, cursor.border, limit+1)
	if err != nil {
		return yacycrawlcontract.CrawlPage{}, err
	}
	shown := crawls[max(len(crawls)-limit, 0):]
	return crawlPageOf(shown, true, len(crawls) > limit), nil
}

func crawlPageOf(
	crawls []yacycrawlcontract.Crawl,
	olderCrawlsBeyond bool,
	newerCrawlsBeyond bool,
) yacycrawlcontract.CrawlPage {
	page := yacycrawlcontract.CrawlPage{Crawls: crawls}
	if len(crawls) == 0 {
		return page
	}
	if olderCrawlsBeyond {
		page.OlderCursor = cursorFrom(olderCursorWord, crawls[len(crawls)-1])
	}
	if newerCrawlsBeyond {
		page.NewerCursor = cursorFrom(newerCursorWord, crawls[0])
	}
	return page
}

func cursorFrom(direction string, crawl yacycrawlcontract.Crawl) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strings.Join([]string{
		direction, crawl.CrawlID,
	}, cursorBoundary)))
}
