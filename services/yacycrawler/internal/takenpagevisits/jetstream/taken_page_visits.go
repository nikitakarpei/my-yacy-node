package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/jetstreamrecord"
)

const BucketName = "YACY_TAKEN_PAGE_VISITS"

func Ensure(
	ctx context.Context,
	js jetstream.JetStream,
	spec jetstreamrecord.BucketSpec,
) error {
	return jetstreamrecord.EnsureBucket(ctx, js, BucketName, spec)
}

type TakenPageVisits struct {
	pageVisits *jetstreamrecord.Records[takenPageVisit]
}

func New(bucket jetstream.KeyValue) *TakenPageVisits {
	return &TakenPageVisits{pageVisits: jetstreamrecord.New[takenPageVisit](bucket)}
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
	return jetstreamrecord.KeyOf(orderID, url.String())
}
