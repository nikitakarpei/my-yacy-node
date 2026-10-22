package orderintake_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/pullintaketest"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/acceptedorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/orderintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

var keptCrawlID = crawlid.CrawlID{1}

type fakeCrawls struct {
	kept []acceptedorder.AcceptedOrder
	err  error
}

func (o *fakeCrawls) Keep(
	_ *vault.Txn,
	order acceptedorder.AcceptedOrder,
) (crawlid.CrawlID, error) {
	if o.err != nil {
		return crawlid.CrawlID{}, o.err
	}
	o.kept = append(o.kept, order)
	return keptCrawlID, nil
}

type fakePendingVisits struct {
	published    []pendingpagevisit.PendingPageVisit
	publications int
	err          error
}

func (v *fakePendingVisits) Owe(
	_ *vault.Txn,
	visits []pendingpagevisit.PendingPageVisit,
) error {
	if v.err != nil {
		return v.err
	}
	v.publications++
	v.published = append(v.published, visits...)
	return nil
}

type recordingObserver struct {
	accepted       int
	acceptedCrawls []crawlid.CrawlID
	rejected       int
	returned       int
}

func (o *recordingObserver) CrawlOrderAccepted(
	_ context.Context,
	_ string,
	crawlID crawlid.CrawlID,
	_ int,
) {
	o.accepted++
	o.acceptedCrawls = append(o.acceptedCrawls, crawlID)
}

func (o *recordingObserver) CrawlOrderRejected(context.Context, string, string, error) {
	o.rejected++
}

func (o *recordingObserver) CrawlOrderReturned(context.Context, string, error) {
	o.returned++
}

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
	orders *fakeCrawls,
	frontier *fakePendingVisits,
	observer *recordingObserver,
) error {
	t.Helper()
	storage, err := vault.New(memoryvault.OpenEngine(0), nil, nil, vaultenginetest.FrozenClock{})
	if err != nil {
		t.Fatalf("vault: %v", err)
	}
	return orderintake.NewOrderConsumer(
		storage,
		pullintaketest.MessageSourceOf(message),
		orders,
		frontier,
		observer,
		1,
	).Run(context.Background())
}

func TestAnAcceptedOrderSeedsTheFrontierThenAcknowledges(t *testing.T) {
	message := orderMessage(t, domainProfile())
	orders, visits, observer := &fakeCrawls{}, &fakePendingVisits{}, &recordingObserver{}

	if err := consume(t, message, orders, visits, observer); err != nil {
		t.Fatalf("run: %v", err)
	}

	if len(orders.kept) != 1 {
		t.Fatalf("kept %d orders, want 1", len(orders.kept))
	}
	if len(visits.published) != 2 || visits.publications != 1 {
		t.Fatalf(
			"published %d seeds in %d publications, want 2 in one",
			len(visits.published),
			visits.publications,
		)
	}
	for _, visit := range visits.published {
		if visit.CrawlID != keptCrawlID || visit.Depth != 0 {
			t.Fatalf("seed %+v should carry its crawl at depth zero", visit)
		}
	}
	if got := message.Settlements(); len(got) != 1 || got[0] != pullintaketest.Acknowledged {
		t.Fatalf("message settled %v, want one ack", got)
	}
	if observer.accepted != 1 || observer.acceptedCrawls[0] != keptCrawlID {
		t.Fatalf("observer %+v, want one accepted order of crawl %s", observer, keptCrawlID)
	}
}

func TestAnOrderTheCrawlerCannotAcceptReturnsForRedelivery(t *testing.T) {
	message := orderMessage(t, domainProfile())
	orders := &fakeCrawls{err: errors.New("bucket down")}
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

	if err := consume(t, message, &fakeCrawls{}, visits, observer); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := message.Settlements(); len(got) != 1 || got[0] != pullintaketest.HeldBack {
		t.Fatalf("message settled %v, want one delayed return", got)
	}
	if observer.returned != 1 {
		t.Fatalf("observer %+v, want one redelivery", observer)
	}
}

func TestAnOrderWhoseProfileTheCrawlerCannotReadIsRejectedAndAcknowledged(t *testing.T) {
	profile := domainProfile()
	profile.URLMustNotMatch = "([unclosed"
	message := orderMessage(t, profile)
	orders, visits, observer := &fakeCrawls{}, &fakePendingVisits{}, &recordingObserver{}

	if err := consume(t, message, orders, visits, observer); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := message.Settlements(); len(got) != 1 || got[0] != pullintaketest.Acknowledged {
		t.Fatalf("message settled %v, want one ack", got)
	}
	if len(orders.kept) != 0 || len(visits.published) != 0 {
		t.Fatal("an order the crawler cannot read should neither be kept nor seed the frontier")
	}
	if observer.rejected != 1 || observer.returned != 0 {
		t.Fatalf("observer %+v, want one rejection", observer)
	}
}

func TestAnUndecodableOrderIsRejectedAndAcknowledged(t *testing.T) {
	message := &pullintaketest.Message{Body: []byte("{")}
	observer := &recordingObserver{}

	if err := consume(
		t, message, &fakeCrawls{}, &fakePendingVisits{}, observer,
	); err != nil {
		t.Fatalf("run: %v", err)
	}

	if got := message.Settlements(); len(got) != 1 || got[0] != pullintaketest.Acknowledged {
		t.Fatalf("message settled %v, want one ack", got)
	}
	if observer.rejected != 1 {
		t.Fatalf("observer %+v, want one rejection", observer)
	}
}
