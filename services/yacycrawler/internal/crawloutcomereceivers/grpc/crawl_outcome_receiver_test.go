package grpc_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	crawlerv1 "github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/crawler/v1"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/crawloutcomesclienttest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/disposal"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawloutcomereceivers/grpc"
)

const (
	serveTimeLimit = 10 * time.Second
	retryPause     = 50 * time.Millisecond
	readURL        = "https://example.com/"
	resolvedURL    = "https://example.com/moved"
)

var errBucketUnreachable = errors.New("bucket unreachable")

type recordedResolutions struct {
	byCanonicalURL map[canonicalurl.CanonicalURL]canonicalurl.CanonicalURL
	failure        error
}

func (r recordedResolutions) ResolvedURLOf(
	_ context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) (canonicalurl.CanonicalURL, error) {
	if r.failure != nil {
		return canonicalurl.CanonicalURL{}, r.failure
	}
	resolved, recorded := r.byCanonicalURL[canonicalURL]
	if !recorded {
		return canonicalURL, nil
	}
	return resolved, nil
}

type recordedDisposals struct {
	byCanonicalURL map[canonicalurl.CanonicalURL]disposal.DisposedPage
	failure        error
}

func (d recordedDisposals) DisposedPageOf(
	_ context.Context,
	canonicalURL canonicalurl.CanonicalURL,
) (disposal.DisposedPage, bool, error) {
	if d.failure != nil {
		return disposal.DisposedPage{}, false, d.failure
	}
	disposedPage, disposed := d.byCanonicalURL[canonicalURL]
	return disposedPage, disposed, nil
}

type receiverUnderTest struct {
	t      *testing.T
	client crawlerv1.CrawlOutcomesClient
}

func crawlOutcomeReceiverUnderTest(
	t *testing.T,
	redirectResolutions grpc.RedirectResolutions,
	disposedPages grpc.DisposedPages,
) receiverUnderTest {
	t.Helper()
	listenAddress := freeListenAddress(t)
	receiver := grpc.NewCrawlOutcomeReceiver(redirectResolutions, disposedPages, listenAddress)
	ctx, cancel := context.WithCancel(context.Background())
	served := make(chan error, 1)
	go func() { served <- receiver.Serve(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-served:
			if err != nil {
				t.Errorf("serve: %v", err)
			}
		case <-time.After(serveTimeLimit):
			t.Error("receiver did not stop after cancel")
		}
	})
	waitUntilListening(t, listenAddress)
	return receiverUnderTest{
		t:      t,
		client: crawloutcomesclienttest.New(t, listenAddress),
	}
}

func (r receiverUnderTest) ReadPage(url string) (*crawlerv1.PageOutcome, error) {
	r.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), serveTimeLimit)
	defer cancel()
	return r.client.ReadPage(ctx, &crawlerv1.ReadPageRequest{Url: url})
}

func freeListenAddress(t *testing.T) string {
	t.Helper()
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve address: %v", err)
	}
	listenAddress := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("release address: %v", err)
	}
	return listenAddress
}

func waitUntilListening(t *testing.T, listenAddress string) {
	t.Helper()
	var dialer net.Dialer
	deadline := time.Now().Add(serveTimeLimit)
	for time.Now().Before(deadline) {
		connection, err := dialer.DialContext(context.Background(), "tcp", listenAddress)
		if err == nil {
			_ = connection.Close()
			return
		}
		time.Sleep(retryPause)
	}
	t.Fatalf("receiver never listened at %s", listenAddress)
}

func TestAReceiverRefusesToServeAListenAddressThatCannotBind(t *testing.T) {
	receiver := grpc.NewCrawlOutcomeReceiver(
		recordedResolutions{}, recordedDisposals{}, "127.0.0.1:99999",
	)

	if err := receiver.Serve(context.Background()); err == nil {
		t.Fatal("expected error when listen address cannot bind")
	}
}

func TestReadPageAnswersWithTheURLCrawlingResolvedThePageTo(t *testing.T) {
	receiver := crawlOutcomeReceiverUnderTest(t, recordedResolutions{
		byCanonicalURL: map[canonicalurl.CanonicalURL]canonicalurl.CanonicalURL{
			canonicalurltest.CanonicalURLOf(t, readURL): canonicalurltest.CanonicalURLOf(
				t, resolvedURL,
			),
		},
	}, recordedDisposals{})

	outcome, err := receiver.ReadPage(readURL)
	if err != nil {
		t.Fatalf("read page: %v", err)
	}

	if outcome.GetCanonicalUrl() != readURL {
		t.Errorf("canonicalUrl = %q, want %q", outcome.GetCanonicalUrl(), readURL)
	}
	if outcome.GetResolvedUrl() != resolvedURL {
		t.Errorf("resolvedUrl = %q, want %q", outcome.GetResolvedUrl(), resolvedURL)
	}
	if outcome.GetDisposal() != nil {
		t.Errorf("disposal = %v, want none", outcome.GetDisposal())
	}
}

func TestReadPageAnswersWithThePageItselfWhenCrawlingRecordedNoRedirect(t *testing.T) {
	receiver := crawlOutcomeReceiverUnderTest(t, recordedResolutions{}, recordedDisposals{})

	outcome, err := receiver.ReadPage(readURL)
	if err != nil {
		t.Fatalf("read page: %v", err)
	}

	if outcome.GetResolvedUrl() != readURL {
		t.Errorf("resolvedUrl = %q, want %q", outcome.GetResolvedUrl(), readURL)
	}
}

func TestReadPageAnswersWithTheMarkAndReasonOfADisposedPage(t *testing.T) {
	receiver := crawlOutcomeReceiverUnderTest(t, recordedResolutions{}, recordedDisposals{
		byCanonicalURL: map[canonicalurl.CanonicalURL]disposal.DisposedPage{
			canonicalurltest.CanonicalURLOf(t, readURL): {
				Mark:   "00000000000000000007",
				Reason: disposal.Oversized,
			},
		},
	})

	outcome, err := receiver.ReadPage(readURL)
	if err != nil {
		t.Fatalf("read page: %v", err)
	}

	if outcome.GetDisposal().GetMark() != "00000000000000000007" {
		t.Errorf("mark = %q, want %q", outcome.GetDisposal().GetMark(), "00000000000000000007")
	}
	if outcome.GetDisposal().GetReason() != string(disposal.Oversized) {
		t.Errorf("reason = %q, want %q", outcome.GetDisposal().GetReason(), disposal.Oversized)
	}
}

func TestReadPageRefusesARequestWhoseURLIsNotCanonicalizable(t *testing.T) {
	receiver := crawlOutcomeReceiverUnderTest(t, recordedResolutions{}, recordedDisposals{})

	_, err := receiver.ReadPage("")

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestReadPageReportsABucketFailureAsInternal(t *testing.T) {
	receiver := crawlOutcomeReceiverUnderTest(
		t,
		recordedResolutions{failure: errBucketUnreachable},
		recordedDisposals{},
	)

	_, err := receiver.ReadPage(readURL)

	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.Internal)
	}
}
