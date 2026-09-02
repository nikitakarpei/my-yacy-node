package jetstream

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

type ScrapeRequestsStreamLimits struct {
	MaxMsgs int64
}

type ScrapePageOffersStreamLimits struct {
	MaxBytes int64
	MaxAge   time.Duration
}

func CreateScrapeRequestsStream(
	ctx context.Context,
	broker jetstream.JetStream,
	limits ScrapeRequestsStreamLimits,
) (jetstream.Stream, error) {
	stream, err := broker.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name: pagescrapecontract.ScrapeRequestsStreamName,
		Subjects: []string{
			pagescrapecontract.ScrapeRequestSubject,
			pagescrapecontract.EveryScrapeScheduleSubject,
		},
		Retention:         jetstream.LimitsPolicy,
		Discard:           jetstream.DiscardOld,
		MaxMsgs:           limits.MaxMsgs,
		AllowMsgSchedules: true,
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
) error {
	if _, err := broker.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name: pagescrapecontract.ScrapePageOffersStreamName,
		Subjects: []string{
			pagescrapecontract.OfferedPageSubject,
			pagescrapecontract.ScrapeFailureSubject,
		},
		Retention: jetstream.InterestPolicy,
		Discard:   jetstream.DiscardOld,
		MaxBytes:  limits.MaxBytes,
		MaxAge:    limits.MaxAge,
	}); err != nil {
		return fmt.Errorf(
			"create the %s stream: %w", pagescrapecontract.ScrapePageOffersStreamName, err,
		)
	}
	return nil
}
