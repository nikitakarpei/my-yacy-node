package jetstream_test

import (
	"context"
	"testing"
	"time"

	natsjetstream "github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/pendingvisits/jetstream"
)

func frontier(t *testing.T) (natsjetstream.JetStream, *jetstream.Publisher) {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	if _, err := js.CreateOrUpdateStream(context.Background(), natsjetstream.StreamConfig{
		Name:       pendingvisit.StreamName,
		Subjects:   []string{pendingvisit.Subject},
		Retention:  natsjetstream.WorkQueuePolicy,
		Duplicates: time.Minute,
	}); err != nil {
		t.Fatal(err)
	}
	return js, jetstream.New(js)
}

func pendingVisits(t *testing.T, js natsjetstream.JetStream) uint64 {
	t.Helper()
	stream, err := js.Stream(context.Background(), pendingvisit.StreamName)
	if err != nil {
		t.Fatal(err)
	}
	info, err := stream.Info(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return info.State.Msgs
}

func TestPublishPutsThePendingVisitOnTheFrontier(t *testing.T) {
	js, publisher := frontier(t)
	ctx := context.Background()
	visit := pendingvisit.PendingVisit{
		OrderID: "o1",
		URL:     canonicalurltest.CanonicalURLOf(t, "http://host/page"),
		Depth:   1,
	}

	if err := publisher.Publish(ctx, visit); err != nil {
		t.Fatalf("publish: %v", err)
	}

	stream, err := js.Stream(ctx, pendingvisit.StreamName)
	if err != nil {
		t.Fatal(err)
	}
	consumer, err := stream.CreateOrUpdateConsumer(ctx, natsjetstream.ConsumerConfig{
		AckPolicy: natsjetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatal(err)
	}
	msg, err := consumer.Next(natsjetstream.FetchMaxWait(5 * time.Second))
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	_ = msg.Ack()

	read, err := pendingvisit.UnmarshalPendingVisit(msg.Data())
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if read != visit {
		t.Fatalf("read %+v, want %+v", read, visit)
	}
}

func TestPublishTheSameURLTwiceLeavesOneMessage(t *testing.T) {
	js, publisher := frontier(t)
	ctx := context.Background()
	visit := pendingvisit.PendingVisit{
		OrderID: "o1",
		URL:     canonicalurltest.CanonicalURLOf(t, "http://host/page"),
		Depth:   1,
	}

	for range 3 {
		if err := publisher.Publish(ctx, visit); err != nil {
			t.Fatalf("publish: %v", err)
		}
	}

	if pending := pendingVisits(t, js); pending != 1 {
		t.Fatalf("frontier holds %d messages, want 1", pending)
	}
}

func TestPublishTheSameURLForAnotherOrderKeepsBoth(t *testing.T) {
	js, publisher := frontier(t)
	ctx := context.Background()
	url := canonicalurltest.CanonicalURLOf(t, "http://host/page")

	for _, orderID := range []string{"o1", "o2"} {
		if err := publisher.Publish(ctx, pendingvisit.PendingVisit{
			OrderID: orderID, URL: url, Depth: 0,
		}); err != nil {
			t.Fatalf("publish: %v", err)
		}
	}

	if pending := pendingVisits(t, js); pending != 2 {
		t.Fatalf("frontier holds %d messages, want 2", pending)
	}
}
