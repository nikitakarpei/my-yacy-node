package companionrequests

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
)

func answerUntilDone(
	ctx context.Context,
	connection *nats.Conn,
	handlers map[string]nats.MsgHandler,
) error {
	subscriptions := make([]*nats.Subscription, 0, len(handlers))
	for subject, handler := range handlers {
		subscription, err := connection.Subscribe(subject, handler)
		if err != nil {
			return fmt.Errorf("subscribe to %q: %w", subject, err)
		}
		subscriptions = append(subscriptions, subscription)
	}
	<-ctx.Done()
	for _, subscription := range subscriptions {
		if err := subscription.Unsubscribe(); err != nil {
			slog.WarnContext(ctx, "companion requests not unsubscribed",
				slog.String("subject", subscription.Subject), slog.Any("error", err),
			)
		}
	}
	return nil
}

func respond(ctx context.Context, message *nats.Msg, reply any) {
	data, err := json.Marshal(reply)
	if err == nil {
		err = message.Respond(data)
	}
	if err != nil {
		slog.WarnContext(ctx, "companion reply not sent",
			slog.String("subject", message.Subject), slog.Any("error", err),
		)
	}
}
