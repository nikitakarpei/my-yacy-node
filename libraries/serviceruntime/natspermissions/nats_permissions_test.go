package natspermissions_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/natspermissions"
)

const (
	ordersStream   = "orders"
	newOrder       = "orders.new"
	workerDurable  = "worker"
	otherDurable   = "other"
	everyInbox     = "_INBOX.>"
	violationWait  = 5 * time.Second
	messageWait    = 5 * time.Second
	unlimitedSetup = 5 * time.Second
)

func TestPullingFromLetsAServiceTakeAndAcknowledgeTheMessagesOfItsConsumer(t *testing.T) {
	limited, _ := limitedJetStreamWith(t, natspermissions.JoinedFrom(
		natspermissions.PullingFrom(ordersStream, workerDurable),
		natspermissions.SubscribingTo(everyInbox),
	))

	consumer := consumerOf(t, limited, workerDurable)
	message, err := consumer.Next(jetstream.FetchMaxWait(messageWait))
	if err != nil {
		t.Fatalf("next message: %v", err)
	}
	if err := message.DoubleAck(t.Context()); err != nil {
		t.Errorf("acknowledge: %v", err)
	}
}

func TestPullingFromRefusesTheMessagesOfAnotherConsumer(t *testing.T) {
	limited, violations := limitedJetStreamWith(t, natspermissions.JoinedFrom(
		natspermissions.PullingFrom(ordersStream, workerDurable),
		natspermissions.SubscribingTo(everyInbox),
	))
	stream, err := limited.Stream(t.Context(), ordersStream)
	if err != nil {
		t.Fatalf("open the stream: %v", err)
	}

	go func() { _, _ = stream.Consumer(context.Background(), otherDurable) }()

	requireViolation(t, violations)
}

func TestWatchingBacklogOfLetsAServiceListTheConsumersOfAStream(t *testing.T) {
	limited, _ := limitedJetStreamWith(t, natspermissions.JoinedFrom(
		natspermissions.WatchingBacklogOf(ordersStream),
		natspermissions.SubscribingTo(everyInbox),
	))
	stream, err := limited.Stream(t.Context(), ordersStream)
	if err != nil {
		t.Fatalf("open the stream: %v", err)
	}

	var durables []string
	consumers := stream.ListConsumers(t.Context())
	for info := range consumers.Info() {
		durables = append(durables, info.Config.Durable)
	}
	if err := consumers.Err(); err != nil {
		t.Fatalf("list consumers: %v", err)
	}
	if !slices.Contains(durables, workerDurable) {
		t.Errorf("consumers = %v, want %s among them", durables, workerDurable)
	}
}

func TestPublishingOnLetsAServiceAddMessagesToAStream(t *testing.T) {
	limited, _ := limitedJetStreamWith(t, natspermissions.JoinedFrom(
		natspermissions.PublishingOn(newOrder),
		natspermissions.SubscribingTo(everyInbox),
	))

	if _, err := limited.Publish(t.Context(), newOrder, []byte("second order")); err != nil {
		t.Errorf("publish: %v", err)
	}
}

func TestJoinedFromKeepsEachSubjectOnce(t *testing.T) {
	joined := natspermissions.JoinedFrom(
		natspermissions.PullingFrom(ordersStream, workerDurable),
		natspermissions.WatchingBacklogOf(ordersStream),
		natspermissions.SubscribingTo(everyInbox),
		natspermissions.SubscribingTo(everyInbox),
	)

	for _, subjects := range [][]string{joined.Publish, joined.Subscribe} {
		if len(slices.Compact(slices.Sorted(slices.Values(subjects)))) != len(subjects) {
			t.Errorf("subjects = %v, want each once", subjects)
		}
	}
}

func limitedJetStreamWith(
	t *testing.T,
	permissions natspermissions.Permissions,
) (jetstream.JetStream, <-chan error) {
	t.Helper()
	urls := natstestserver.StartWithLimitedUser(t, natstestserver.Allowed{
		Publish: permissions.Publish, Subscribe: permissions.Subscribe,
	})
	unlimitedURL, limitedURL := urls.Unlimited, urls.Limited
	provisionOrders(t, natstestserver.ConnectJetStream(t, unlimitedURL))
	violations := make(chan error, 8)
	connection, err := nats.Connect(limitedURL,
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			violations <- err
		}),
	)
	if err != nil {
		t.Fatalf("connect as the limited user: %v", err)
	}
	t.Cleanup(connection.Close)
	limited, err := jetstream.New(connection)
	if err != nil {
		t.Fatalf("jetstream of the limited user: %v", err)
	}
	return limited, violations
}

func provisionOrders(t *testing.T, unlimited jetstream.JetStream) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), unlimitedSetup)
	defer cancel()
	stream, err := unlimited.CreateStream(ctx, jetstream.StreamConfig{
		Name: ordersStream, Subjects: []string{ordersStream + ".>"},
	})
	if err != nil {
		t.Fatalf("create the stream: %v", err)
	}
	for _, durable := range []string{workerDurable, otherDurable} {
		if _, err := stream.CreateConsumer(ctx, jetstream.ConsumerConfig{
			Durable: durable, AckPolicy: jetstream.AckExplicitPolicy,
		}); err != nil {
			t.Fatalf("create consumer %s: %v", durable, err)
		}
	}
	if _, err := unlimited.Publish(ctx, newOrder, []byte("first order")); err != nil {
		t.Fatalf("publish an order: %v", err)
	}
}

func consumerOf(t *testing.T, limited jetstream.JetStream, durable string) jetstream.Consumer {
	t.Helper()
	stream, err := limited.Stream(t.Context(), ordersStream)
	if err != nil {
		t.Fatalf("open the stream: %v", err)
	}
	consumer, err := stream.Consumer(t.Context(), durable)
	if err != nil {
		t.Fatalf("open consumer %s: %v", durable, err)
	}
	return consumer
}

func requireViolation(t *testing.T, violations <-chan error) {
	t.Helper()
	select {
	case err := <-violations:
		if !errors.Is(err, nats.ErrPermissionViolation) {
			t.Errorf("error = %v, want a permissions violation", err)
		}
	case <-time.After(violationWait):
		t.Fatal("the broker allowed another consumer")
	}
}
