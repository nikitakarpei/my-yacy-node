package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/redisrecord"
)

const recordKind = "YACY_TAKEN_PAGE_VISITS"

type TakenPageVisits struct {
	pageVisits *redisrecord.Records[takenPageVisit]
}

func New(client redis.UniversalClient, retention time.Duration) *TakenPageVisits {
	return &TakenPageVisits{pageVisits: redisrecord.New[takenPageVisit](client, retention)}
}

func (visits *TakenPageVisits) TakePageVisit(
	ctx context.Context,
	orderID string,
	url canonicalurl.CanonicalURL,
	taker string,
) (bool, error) {
	standing, took, err := visits.pageVisits.Revise(
		ctx, pageVisitKeyOf(orderID, url), takenBy(taker),
	)
	if err != nil {
		return false, fmt.Errorf("take the page visit to %s: %w", url, err)
	}
	return took || standing.Taker == taker, nil
}

func takenBy(taker string) func(takenPageVisit) (takenPageVisit, bool) {
	return func(pageVisit takenPageVisit) (takenPageVisit, bool) {
		if pageVisit.Taker != "" {
			return pageVisit, false
		}
		pageVisit.Taker = taker
		return pageVisit, true
	}
}

type takenPageVisit struct {
	Taker string `json:"Taker"`
}

func pageVisitKeyOf(orderID string, url canonicalurl.CanonicalURL) string {
	return redisrecord.KeyOf(recordKind, orderID, url.String())
}
