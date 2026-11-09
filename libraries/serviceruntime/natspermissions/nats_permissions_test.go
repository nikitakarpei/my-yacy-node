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
	ordersStream    = "orders"
	newOrderSubject = "orders.new"
	workerConsumer  = "worker"
	otherConsumer   = "other"
	everyInbox      = "_INBOX.>"
	violationWait   = 5 * time.Second
	messageWait     = 5 * time.Second
	setupTimeout    = 5 * time.Second
)

func TestPullingFromLetsAServiceTakeAndAcknowledgeTheMessagesOfItsConsumer(t *testing.T) {
	limitedJetStream, _ := limitedJetStreamWith(t, natspermissions.JoinedFrom(
		natspermissions.PullingFrom(ordersStream, workerConsumer),
		natspermissions.SubscribingTo(everyInbox),
	))

	consumer := consumerOf(t, limitedJetStream, workerConsumer)
	message, err := consumer.Next(jetstream.FetchMaxWait(messageWait))
	if err != nil {
		t.Fatalf("next message: %v", err)
	}
	if err := message.DoubleAck(t.Context()); err != nil {
		t.Errorf("acknowledge: %v", err)
	}
}

func TestPullingFromRefusesTheMessagesOfAnotherConsumer(t *testing.T) {
	limitedJetStream, violations := limitedJetStreamWith(t, natspermissions.JoinedFrom(
		natspermissions.PullingFrom(ordersStream, workerConsumer),
		natspermissions.SubscribingTo(everyInbox),
	))
	stream, err := limitedJetStream.Stream(t.Context(), ordersStream)
	if err != nil {
		t.Fatalf("open the stream: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	consumerErrors := make(chan error, 1)
	go func() {
		_, err := stream.Consumer(ctx, otherConsumer)
		consumerErrors <- err
	}()

	requireViolation(t, violations)
	cancel()
	if err := <-consumerErrors; err == nil {
		t.Error("the consumer of another service opened")
	}
}

func TestWatchingBacklogOfLetsAServiceListTheConsumersOfAStream(t *testing.T) {
	limitedJetStream, _ := limitedJetStreamWith(t, natspermissions.JoinedFrom(
		natspermissions.WatchingBacklogOf(ordersStream),
		natspermissions.SubscribingTo(everyInbox),
	))
	stream, err := limitedJetStream.Stream(t.Context(), ordersStream)
	if err != nil {
		t.Fatalf("open the stream: %v", err)
	}

	var consumerNames []string
	consumers := stream.ListConsumers(t.Context())
	for info := range consumers.Info() {
		consumerNames = append(consumerNames, info.Config.Durable)
	}
	if err := consumers.Err(); err != nil {
		t.Fatalf("list consumers: %v", err)
	}
	if !slices.Contains(consumerNames, workerConsumer) {
		t.Errorf("consumers = %v, want %s among them", consumerNames, workerConsumer)
	}
}

func TestPublishingOnLetsAServiceAddMessagesToAStream(t *testing.T) {
	limitedJetStream, _ := limitedJetStreamWith(t, natspermissions.JoinedFrom(
		natspermissions.PublishingOn(newOrderSubject),
		natspermissions.SubscribingTo(everyInbox),
	))

	if _, err := limitedJetStream.Publish(
		t.Context(),
		newOrderSubject,
		[]byte("second order"),
	); err != nil {
		t.Errorf("publish: %v", err)
	}
}

func limitedJetStreamWith(
	t *testing.T,
	permissions natspermissions.Permissions,
) (jetstream.JetStream, <-chan error) {
	t.Helper()
	urls := natstestserver.StartWithLimitedUser(t, natstestserver.AllowedSubjects{
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
	limitedJetStream, err := jetstream.New(connection)
	if err != nil {
		t.Fatalf("jetstream of the limited user: %v", err)
	}
	return limitedJetStream, violations
}

func provisionOrders(t *testing.T, unlimited jetstream.JetStream) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), setupTimeout)
	defer cancel()
	stream, err := unlimited.CreateStream(ctx, jetstream.StreamConfig{
		Name: ordersStream, Subjects: []string{ordersStream + ".>"},
	})
	if err != nil {
		t.Fatalf("create the stream: %v", err)
	}
	for _, durable := range []string{workerConsumer, otherConsumer} {
		if _, err := stream.CreateConsumer(ctx, jetstream.ConsumerConfig{
			Durable: durable, AckPolicy: jetstream.AckExplicitPolicy,
		}); err != nil {
			t.Fatalf("create consumer %s: %v", durable, err)
		}
	}
	if _, err := unlimited.Publish(ctx, newOrderSubject, []byte("first order")); err != nil {
		t.Fatalf("publish an order: %v", err)
	}
}

func consumerOf(
	t *testing.T,
	limitedJetStream jetstream.JetStream,
	consumerName string,
) jetstream.Consumer {
	t.Helper()
	stream, err := limitedJetStream.Stream(t.Context(), ordersStream)
	if err != nil {
		t.Fatalf("open the stream: %v", err)
	}
	consumer, err := stream.Consumer(t.Context(), consumerName)
	if err != nil {
		t.Fatalf("open consumer %s: %v", consumerName, err)
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
