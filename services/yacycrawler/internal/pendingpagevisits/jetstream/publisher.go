// Package jetstream puts every URL the crawler admits on the frontier stream.
package jetstream

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

type Publisher struct {
	stream jetstream.JetStream
}

func New(stream jetstream.JetStream) *Publisher {
	return &Publisher{stream: stream}
}

func (p *Publisher) Publish(
	ctx context.Context,
	pageVisit pendingpagevisit.PendingPageVisit,
) error {
	data, err := pendingpagevisit.MarshalPendingPageVisit(pageVisit)
	if err != nil {
		return err
	}
	if _, err := p.stream.Publish(
		ctx,
		pendingpagevisit.Subject,
		data,
		jetstream.WithMsgID(messageIdentityOf(pageVisit)),
	); err != nil {
		return fmt.Errorf("publish pending page pageVisit %s: %w", pageVisit.URL, err)
	}
	return nil
}

func messageIdentityOf(pageVisit pendingpagevisit.PendingPageVisit) string {
	return digestOf(pageVisit.OrderID) + "." + digestOf(pageVisit.URL.String())
}

func digestOf(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
