// Package jetstream puts every URL the crawler admits on the frontier stream.
package jetstream

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
)

type Publisher struct {
	stream jetstream.JetStream
}

func New(stream jetstream.JetStream) *Publisher {
	return &Publisher{stream: stream}
}

func (p *Publisher) Publish(ctx context.Context, visit pendingvisit.PendingVisit) error {
	data, err := pendingvisit.MarshalPendingVisit(visit)
	if err != nil {
		return err
	}
	if _, err := p.stream.Publish(
		ctx,
		pendingvisit.Subject,
		data,
		jetstream.WithMsgID(messageIdentityOf(visit)),
	); err != nil {
		return fmt.Errorf("publish pending visit %s: %w", visit.URL, err)
	}
	return nil
}

func messageIdentityOf(visit pendingvisit.PendingVisit) string {
	return digestOf(visit.OrderID) + "." + digestOf(visit.URL.String())
}

func digestOf(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
