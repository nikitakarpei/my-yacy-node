package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

type ScrapeRequestsStreamLimits struct {
	MaxMsgs int64
}

type ScrapePageOffersStreamLimits struct {
	MaxBytes int64
}

func CreateScrapeRequestsStream(
	ctx context.Context,
	broker jetstream.JetStream,
	limits ScrapeRequestsStreamLimits,
) (jetstream.Stream, error) {
	stream, err := broker.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      pagescrapecontract.ScrapeRequestsStreamName,
		Subjects:  []string{pagescrapecontract.ScrapeRequestSubject},
		Retention: jetstream.WorkQueuePolicy,
		Discard:   jetstream.DiscardNew,
		MaxMsgs:   limits.MaxMsgs,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"create the %s stream: %w", pagescrapecontract.ScrapeRequestsStreamName, err,
		)
	}
	return stream, nil
}

func CreateScrapePageOffersStream(
	ctx context.Context,
	broker jetstream.JetStream,
	limits ScrapePageOffersStreamLimits,
) (jetstream.Stream, error) {
	stream, err := broker.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name: pagescrapecontract.ScrapePageOffersStreamName,
		Subjects: []string{
			pagescrapecontract.OfferedPageSubject,
			pagescrapecontract.ScrapeFailureSubject,
		},
		Retention: jetstream.InterestPolicy,
		Discard:   jetstream.DiscardNew,
		MaxBytes:  limits.MaxBytes,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"create the %s stream: %w", pagescrapecontract.ScrapePageOffersStreamName, err,
		)
	}
	return stream, nil
}
