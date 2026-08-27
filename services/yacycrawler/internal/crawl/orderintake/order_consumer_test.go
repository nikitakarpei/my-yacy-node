package orderintake_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/pullintaketest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/orderintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
)

type fakeAcceptedOrders struct {
	kept []acceptedorder.AcceptedOrder
	err  error
}

func (o *fakeAcceptedOrders) Keep(_ context.Context, order acceptedorder.AcceptedOrder) error {
	if o.err != nil {
		return o.err
	}
	o.kept = append(o.kept, order)
	return nil
}

type fakePendingVisits struct {
	published []pendingvisit.PendingVisit
	err       error
}

func (v *fakePendingVisits) Publish(_ context.Context, visit pendingvisit.PendingVisit) error {
	if v.err != nil {
		return v.err
	}
	v.published = append(v.published, visit)
	return nil
}

type recordingObserver struct {
	received int
	accepted int
	returned int
}

func (o *recordingObserver) OrderReceived() { o.received++ }
func (o *recordingObserver) OrderAccepted() { o.accepted++ }
func (o *recordingObserver) OrderReturned() { o.returned++ }

func seeds(t *testing.T) []canonicalurl.CanonicalURL {
	t.Helper()
	return []canonicalurl.CanonicalURL{
		canonicalurltest.CanonicalURLOf(t, "http://host/one"),
		canonicalurltest.CanonicalURLOf(t, "http://host/two"),
	}
}

func domainProfile() yacycrawlcontract.CrawlProfile {
	return yacycrawlcontract.CrawlProfile{
		Scope:           yacycrawlcontract.ScopeDomain,
		URLMustMatch:    yacycrawlcontract.MatchAll,
		MaxPagesPerHost: yacycrawlcontract.UnlimitedPagesPerHost,
	}
}

func orderMessage(
	t *testing.T,
	profile yacycrawlcontract.CrawlProfile,
) *pullintaketest.Message {
	t.Helper()
	data, err := yacycrawlcontract.MarshalCrawlOrder(yacycrawlcontract.CrawlOrder{
		OrderID:  "o1",
		Profile:  profile,
		SeedURLs: seeds(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	return &pullintaketest.Message{Body: data}
}

func consume(
	t *testing.T,
	message jetstream.Msg,
	orders *fakeAcceptedOrders,
	visits *fakePendingVisits,
	observer *recordingObserver,
) error {
	t.Helper()
	return orderintake.NewOrderConsumer(orderintake.Config{
		Source:                 pullintaketest.MessageSourceOf(message),
		Orders:                 orders,
		Visits:                 visits,
		Observer:               observer,
		OrderIntakeConcurrency: 1,
	}).Run(context.Background())
}

func TestAnAcceptedOrderSeedsTheFrontierThenAcknowledges(t *testing.T) {
	message := orderMessage(t, domainProfile())
	orders, visits, observer := &fakeAcceptedOrders{}, &fakePendingVisits{}, &recordingObserver{}

	if err := consume(t, message, orders, visits, observer); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(orders.kept) != 1 {
		t.Fatalf("kept %d orders, want 1", len(orders.kept))
	}
	if len(visits.published) != 2 {
		t.Fatalf("published %d seeds, want 2", len(visits.published))
	}
	for _, visit := range visits.published {
		if visit.OrderID != "o1" || visit.Depth != 0 {
			t.Fatalf("seed %+v should carry its order at depth zero", visit)
		}
	}
	if got := message.Settlements(); len(got) != 1 || got[0] != pullintaketest.Acknowledged {
		t.Fatalf("message settled %v, want one ack", got)
	}
	if observer.received != 1 || observer.accepted != 1 {
		t.Fatalf("observer %+v, want one received and one accepted", observer)
	}
}

func TestAnOrderTheCrawlerCannotAcceptReturnsForRedelivery(t *testing.T) {
	message := orderMessage(t, domainProfile())
	orders := &fakeAcceptedOrders{err: errors.New("bucket down")}
	visits, observer := &fakePendingVisits{}, &recordingObserver{}

	if err := consume(t, message, orders, visits, observer); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := message.Settlements(); len(got) != 1 || got[0] != pullintaketest.HeldBack {
		t.Fatalf("message settled %v, want one delayed return", got)
	}
	if len(visits.published) != 0 {
		t.Fatal("an order that was not accepted should seed nothing")
	}
	if observer.returned != 1 || observer.accepted != 0 {
		t.Fatalf("observer %+v, want one redelivery", observer)
	}
}

func TestAnOrderWhoseSeedsDoNotPublishReturnsForRedelivery(t *testing.T) {
	message := orderMessage(t, domainProfile())
	visits := &fakePendingVisits{err: errors.New("stream down")}
	observer := &recordingObserver{}

	if err := consume(t, message, &fakeAcceptedOrders{}, visits, observer); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := message.Settlements(); len(got) != 1 || got[0] != pullintaketest.HeldBack {
		t.Fatalf("message settled %v, want one delayed return", got)
	}
	if observer.returned != 1 {
		t.Fatalf("observer %+v, want one redelivery", observer)
	}
}

func TestAnOrderWhoseProfileTheCrawlerCannotReadSeedsNothing(t *testing.T) {
	profile := domainProfile()
	profile.URLMustNotMatch = "([unclosed"
	message := orderMessage(t, profile)
	orders, visits, observer := &fakeAcceptedOrders{}, &fakePendingVisits{}, &recordingObserver{}

	if err := consume(t, message, orders, visits, observer); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := message.Settlements(); len(got) != 1 || got[0] != pullintaketest.HeldBack {
		t.Fatalf("message settled %v, want one delayed return", got)
	}
	if len(orders.kept) != 0 || len(visits.published) != 0 {
		t.Fatal("an order the crawler cannot read should neither be accepted nor seed the frontier")
	}
	if observer.returned != 1 {
		t.Fatalf("observer %+v, want one redelivery", observer)
	}
}

func TestAnUndecodableOrderHaltsIntake(t *testing.T) {
	err := consume(
		t, &pullintaketest.Message{Body: []byte("{")}, &fakeAcceptedOrders{},
		&fakePendingVisits{}, &recordingObserver{},
	)

	if err == nil {
		t.Fatal("an undecodable order should halt intake")
	}
}
