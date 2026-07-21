package crawlrequest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/crawlrequest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

type fakeStream struct {
	subject string
	payload []byte
	err     error
}

func (f *fakeStream) Publish(
	_ context.Context,
	subject string,
	payload []byte,
	_ ...jetstream.PublishOpt,
) (*jetstream.PubAck, error) {
	f.subject = subject
	f.payload = payload
	if f.err != nil {
		return nil, f.err
	}
	return &jetstream.PubAck{}, nil
}

func TestPlacePublishesCrawlOrderForSeed(t *testing.T) {
	const canonical = "https://example.com/"
	stream := &fakeStream{}
	placement := crawlrequest.NewOrderPlacement(stream, "yacy.crawl.orders")

	if err := placement.Place(context.Background(), canonical); err != nil {
		t.Fatalf("place: %v", err)
	}
	if stream.subject != "yacy.crawl.orders" {
		t.Errorf("subject = %q", stream.subject)
	}
	order, err := yacycrawlcontract.UnmarshalCrawlOrder(stream.payload)
	if err != nil {
		t.Fatalf("unmarshal published order: %v", err)
	}
	if len(order.SeedURLs) != 1 || order.SeedURLs[0] != canonical {
		t.Errorf("seed urls = %v", order.SeedURLs)
	}
	if order.OrderID == "" {
		t.Error("order id empty")
	}
}

func TestPlacePropagatesPublishError(t *testing.T) {
	placement := crawlrequest.NewOrderPlacement(&fakeStream{err: errors.New("no stream")}, "s")

	if err := placement.Place(context.Background(), "https://example.com/"); err == nil {
		t.Fatal("expected publish error")
	}
}
