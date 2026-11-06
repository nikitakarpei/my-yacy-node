package jetstream

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
)

type PageRWIsStreamLimits struct {
	MaxBytes int64
}

func CreatePageRWIsStream(
	ctx context.Context,
	broker jetstream.JetStream,
	limits PageRWIsStreamLimits,
) (jetstream.Stream, error) {
	stream, err := broker.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      pagerwicontract.PageRWIsStreamName,
		Subjects:  []string{pagerwicontract.PageRWISubject},
		Retention: jetstream.InterestPolicy,
		Discard:   jetstream.DiscardNew,
		MaxBytes:  limits.MaxBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("create the %s stream: %w", pagerwicontract.PageRWIsStreamName, err)
	}
	return stream, nil
}
