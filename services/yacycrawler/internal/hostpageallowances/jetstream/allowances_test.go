package jetstream_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/hostpageallowances/jetstream"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/jetstreamrecord"
)

const (
	orderID = "o1"
	host    = "host"
)

func hostPageAllowances(t *testing.T) *jetstream.Allowances {
	t.Helper()
	js := natstestserver.ConnectJetStream(t, natstestserver.Start(t))
	ctx := context.Background()
	if err := jetstream.Ensure(ctx, js, jetstreamrecord.BucketSpec{}); err != nil {
		t.Fatalf("ensure bucket: %v", err)
	}
	bucket, err := js.KeyValue(ctx, jetstream.BucketName)
	if err != nil {
		t.Fatalf("open bucket: %v", err)
	}
	return jetstream.New(bucket)
}

func pageURL(t *testing.T) canonicalurl.CanonicalURL {
	t.Helper()
	return canonicalurltest.CanonicalURLOf(t, "http://host/page")
}

func holdsHostPage(
	t *testing.T,
	allowances *jetstream.Allowances,
	url canonicalurl.CanonicalURL,
	maxPages int,
) bool {
	t.Helper()
	spent, err := allowances.HoldsHostPage(context.Background(), orderID, url, host, maxPages)
	if err != nil {
		t.Fatalf("spend page: %v", err)
	}
	return spent
}

func TestAHostSpendsNoMorePagesThanItsProfileAllows(t *testing.T) {
	allowances := hostPageAllowances(t)

	for page := range 2 {
		url := canonicalurltest.CanonicalURLOf(t, fmt.Sprintf("http://host/page%d", page))
		if !holdsHostPage(t, allowances, url, 2) {
			t.Fatalf("page %d should be within the host allowance", page+1)
		}
	}

	beyond := canonicalurltest.CanonicalURLOf(t, "http://host/beyond")
	if holdsHostPage(t, allowances, beyond, 2) {
		t.Fatal("a page beyond the host allowance should be refused")
	}
}

func TestAVisitSpendsOneHostPageHoweverOftenItIsResumed(t *testing.T) {
	allowances := hostPageAllowances(t)
	url := pageURL(t)

	for resumption := range 3 {
		if !holdsHostPage(t, allowances, url, 1) {
			t.Fatalf("resumption %d should spend the page the visit already holds", resumption)
		}
	}

	other := canonicalurltest.CanonicalURLOf(t, "http://host/other")
	if holdsHostPage(t, allowances, other, 1) {
		t.Fatal("the resumed visit should have spent the whole allowance of the host")
	}
}

func TestAHostThatAllowsNoPagesSpendsNone(t *testing.T) {
	if holdsHostPage(t, hostPageAllowances(t), pageURL(t), 0) {
		t.Fatal("a host that allows no pages should refuse the first page")
	}
}

func TestAnUnlimitedHostAllowanceSpendsNothing(t *testing.T) {
	allowances := hostPageAllowances(t)

	for range 3 {
		url := pageURL(t)
		if !holdsHostPage(t, allowances, url, yacycrawlcontract.UnlimitedPagesPerHost) {
			t.Fatal("an unlimited host allowance should never refuse a page")
		}
	}
}

func TestTheSameHostSpendsSeparatelyUnderAnotherOrder(t *testing.T) {
	allowances := hostPageAllowances(t)
	url := pageURL(t)
	holdsHostPage(t, allowances, url, 1)

	other := canonicalurltest.CanonicalURLOf(t, "http://host/other")
	spent, err := allowances.HoldsHostPage(context.Background(), "o2", other, host, 1)
	if err != nil {
		t.Fatalf("spend page: %v", err)
	}
	if !spent {
		t.Fatal("another order should hold its own allowance of the host")
	}
}

func TestOnlyOneOfTwoConcurrentVisitsSpendsTheLastPageOfAHost(t *testing.T) {
	allowances := hostPageAllowances(t)
	ctx := context.Background()
	spends := make(chan bool, 2)

	var spending sync.WaitGroup
	for page := range 2 {
		url := canonicalurltest.CanonicalURLOf(t, fmt.Sprintf("http://host/race%d", page))
		spending.Add(1)
		go func() {
			defer spending.Done()
			spent, err := allowances.HoldsHostPage(ctx, orderID, url, host, 1)
			if err != nil {
				t.Errorf("spend page: %v", err)
				return
			}
			spends <- spent
		}()
	}
	spending.Wait()
	close(spends)

	granted := 0
	for spent := range spends {
		if spent {
			granted++
		}
	}
	if granted != 1 {
		t.Fatalf("%d of two concurrent visits spent a host allowance of one, want 1", granted)
	}
}
