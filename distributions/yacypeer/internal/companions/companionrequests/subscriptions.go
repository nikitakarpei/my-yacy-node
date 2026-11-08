package companionrequests

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
)

type Subscriptions struct {
	subscriptions []*nats.Subscription
}

func subscribeTo(
	connection *nats.Conn,
	handlers map[string]nats.MsgHandler,
) (*Subscriptions, error) {
	subscribed := &Subscriptions{}
	for subject, handler := range handlers {
		subscription, err := connection.Subscribe(subject, handler)
		if err != nil {
			return nil, fmt.Errorf("subscribe to %q: %w", subject, err)
		}
		subscribed.subscriptions = append(subscribed.subscriptions, subscription)
	}
	if err := connection.Flush(); err != nil {
		return nil, fmt.Errorf("confirm the companion request subscriptions: %w", err)
	}
	return subscribed, nil
}

func (s *Subscriptions) EndWhenDone(ctx context.Context) {
	<-ctx.Done()
	for _, subscription := range s.subscriptions {
		if err := subscription.Unsubscribe(); err != nil {
			slog.WarnContext(ctx, "companion requests not unsubscribed",
				slog.String("subject", subscription.Subject), slog.Any("error", err),
			)
		}
	}
}
