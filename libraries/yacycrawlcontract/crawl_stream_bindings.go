package yacycrawlcontract

import (
	"context"
	"fmt"
	"strings"

	"github.com/nats-io/nats.go/jetstream"
)

const (
	OrdersStreamName = "YACY_CRAWL_ORDERS"

	crawledPageStreamPrefix = "YACY_CRAWL_PAGE_"
)

func CrawledPageStreamName(representation PageRepresentation) string {
	return crawledPageStreamPrefix + strings.ToUpper(string(representation))
}

type OrdersStreamSpec struct {
	Subject string
}

type CrawledPageStreamSpec struct {
	Subject string
	MaxMsgs int64
}

func EnsureOrdersStream(ctx context.Context, js jetstream.JetStream, spec OrdersStreamSpec) error {
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      OrdersStreamName,
		Subjects:  []string{spec.Subject},
		Retention: jetstream.WorkQueuePolicy,
	}); err != nil {
		return fmt.Errorf("ensure orders stream: %w", err)
	}
	return nil
}

func EnsureCrawledPageStream(
	ctx context.Context,
	js jetstream.JetStream,
	representation PageRepresentation,
	spec CrawledPageStreamSpec,
) error {
	if _, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      CrawledPageStreamName(representation),
		Subjects:  []string{spec.Subject},
		Retention: jetstream.WorkQueuePolicy,
		MaxMsgs:   spec.MaxMsgs,
		Discard:   jetstream.DiscardNew,
	}); err != nil {
		return fmt.Errorf("ensure crawled page stream: %w", err)
	}
	return nil
}
