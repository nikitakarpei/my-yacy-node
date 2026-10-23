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

var startedCrawlID = crawlid.CrawlID{1}

type fakeCrawls struct {
	accepted []acceptedorder.AcceptedOrder
	pending  []pendingpagevisit.PendingPageVisit
	started  bool
	err      error
}

func (a *fakeCrawls) Start(
	_ *vault.Txn,
	order acceptedorder.AcceptedOrder,
) (crawlid.CrawlID, bool, error) {
	if a.err != nil {
		return crawlid.CrawlID{}, false, a.err
	}
	a.accepted = append(a.accepted, order)
	return startedCrawlID, a.started, nil
}

func (a *fakeCrawls) Add(_ *vault.Txn, pageVisits []pendingpagevisit.PendingPageVisit) error {
	a.pending = append(a.pending, pageVisits...)
	return nil
}

type recordingObserver struct {
	accepted        int
	acceptedCrawlID crawlid.CrawlID
	ignored         int
	rejected        int
	returned        int
}

func (o *recordingObserver) CrawlOrderAccepted(
	_ context.Context,
	_ string,
	crawlID crawlid.CrawlID,
	_ int,
) {
	o.accepted++
	o.acceptedCrawlID = crawlID
}

func (o *recordingObserver) CrawlOrderIgnored(context.Context, string) {
	o.ignored++
}

func (o *recordingObserver) CrawlOrderRejected(context.Context, string, string, error) {
	o.rejected++
}

func (o *recordingObserver) CrawlOrderReturned(context.Context, string, error) {
	o.returned++
}

func domainProfile() yacycrawlcontract.CrawlProfile {
	return yacycrawlcontract.CrawlProfile{Scope: yacycrawlcontract.ScopeDomain}
}

func orderMessage(
	t *testing.T,
	profile yacycrawlcontract.CrawlProfile,
	seeds ...canonicalurl.CanonicalURL,
) *pullintaketest.Message {
	t.Helper()
	data, err := yacycrawlcontract.MarshalCrawlOrder(yacycrawlcontract.CrawlOrder{
		OrderID:  "o1",
		Profile:  profile,
		SeedURLs: seeds,
	})
	if err != nil {
		t.Fatal(err)
	}
	return &pullintaketest.Message{Body: data}
}

func seed(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()
	return canonicalurltest.CanonicalURLOf(t, "http://host/one")
}

func consume(
	t *testing.T,
	message jetstream.Msg,
	crawls *fakeCrawls,
	observer *recordingObserver,
) {
	t.Helper()
	storage, err := vault.New(memoryvault.OpenEngine(0), nil, nil, vaultenginetest.FrozenClock{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := orderintake.NewOrderConsumer(
		pullintaketest.MessageSourceOf(message),
		storage,
		crawls,
		crawls,
		observer,
		1,
	).Run(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
}

func requireOneSettlement(
	t *testing.T,
	message *pullintaketest.Message,
	want string,
) {
	t.Helper()
	if got := message.Settlements(); len(got) != 1 || got[0] != want {
		t.Fatalf("message settled %v, want one %v", got, want)
	}
}

func TestAnAcceptedOrderIsAcknowledged(t *testing.T) {
	message := orderMessage(t, domainProfile(), seed(t))
	crawls, observer := &fakeCrawls{started: true}, &recordingObserver{}

	consume(t, message, crawls, observer)

	if len(crawls.accepted) != 1 || crawls.accepted[0].OrderID() != "o1" {
		t.Fatalf("accepted %v, want order o1", crawls.accepted)
	}
	if len(crawls.pending) != 1 || crawls.pending[0].CrawlID != startedCrawlID {
		t.Fatalf("pending %v, want the seed of crawl %s", crawls.pending, startedCrawlID)
	}
	requireOneSettlement(t, message, pullintaketest.Acknowledged)
	if observer.accepted != 1 || observer.acceptedCrawlID != startedCrawlID {
		t.Fatalf("observer %+v, want one accepted order of crawl %s", observer, startedCrawlID)
	}
}

func TestAnOrderWhoseCrawlStartedBeforeIsAcknowledgedAndIgnored(t *testing.T) {
	message := orderMessage(t, domainProfile(), seed(t))
	observer := &recordingObserver{}

	crawls := &fakeCrawls{started: false}

	consume(t, message, crawls, observer)

	requireOneSettlement(t, message, pullintaketest.Acknowledged)
	if observer.ignored != 1 || observer.accepted != 0 || len(crawls.pending) != 0 {
		t.Fatalf("observer %+v, want one ignored order", observer)
	}
}

func TestAnOrderTheCrawlerCannotAcceptYetReturnsForRedelivery(t *testing.T) {
	message := orderMessage(t, domainProfile(), seed(t))
	observer := &recordingObserver{}

	consume(t, message, &fakeCrawls{err: errors.New("server down")}, observer)

	requireOneSettlement(t, message, pullintaketest.HeldBack)
	if observer.returned != 1 || observer.accepted != 0 {
		t.Fatalf("observer %+v, want one redelivery", observer)
	}
}

func TestAnOrderTheCrawlerCannotReadIsRejectedAndAcknowledged(t *testing.T) {
	unreadableProfile := domainProfile()
	unreadableProfile.URLMustNotMatch = "([unclosed"
	for name, message := range map[string]*pullintaketest.Message{
		"undecodable":        {Body: []byte("{")},
		"without seeds":      orderMessage(t, domainProfile()),
		"unreadable profile": orderMessage(t, unreadableProfile, seed(t)),
	} {
		t.Run(name, func(t *testing.T) {
			crawls, observer := &fakeCrawls{started: true}, &recordingObserver{}

			consume(t, message, crawls, observer)

			requireOneSettlement(t, message, pullintaketest.Acknowledged)
			if len(crawls.accepted) != 0 {
				t.Fatal("an order the crawler cannot read should not be accepted")
			}
			if observer.rejected != 1 {
				t.Fatalf("observer %+v, want one rejection", observer)
			}
		})
	}
}
