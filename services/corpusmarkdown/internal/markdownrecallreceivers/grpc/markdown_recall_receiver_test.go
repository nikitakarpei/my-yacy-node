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
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecall"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/markdownrecallreceivers/grpc"
	corpusmarkdownv1 "github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore/corpusmarkdown/v1"
	"github.com/nikitakarpei/yacy-rwi-node/pagemarkdownstore/markdowncorpusclienttest"
)

const (
	serveTimeLimit = 10 * time.Second
	retryPause     = 50 * time.Millisecond
	recalledURL    = "https://example.com/"
)

var errCorpusUnreachable = errors.New("corpus unreachable")

var storedAt = time.Date(2026, time.August, 25, 10, 30, 0, 0, time.UTC)

var heldMarkdown = markdownrecall.StoredMarkdown{
	Markdown: []byte("# Hi"),
	StoredAt: storedAt,
	Version:  "SHA-256=jbHfCoRuBAP7Kzb9EGMSJnEcVYYnvKvJcYCA1LMKGVw=",
}

type recalledPages struct {
	byRequestedURL map[canonicalurl.CanonicalURL]markdownrecall.RecalledPage
	failure        error
}

func (r recalledPages) PageOf(
	_ context.Context,
	requestedURL canonicalurl.CanonicalURL,
) (markdownrecall.RecalledPage, bool, error) {
	if r.failure != nil {
		return markdownrecall.RecalledPage{}, false, r.failure
	}
	page, held := r.byRequestedURL[requestedURL]
	if !held {
		return markdownrecall.RecalledPage{}, false, nil
	}
	return page, true, nil
}

type receiverUnderTest struct {
	t      *testing.T
	client corpusmarkdownv1.MarkdownCorpusClient
}

func markdownRecallReceiverUnderTest(
	t *testing.T,
	recall grpc.PageMarkdownRecall,
) receiverUnderTest {
	t.Helper()
	listenAddress := freeListenAddress(t)
	receiver := grpc.NewMarkdownRecallReceiver(recall, listenAddress)
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
		client: markdowncorpusclienttest.New(t, listenAddress),
	}
}

func (r receiverUnderTest) RecallPage(url string) (*corpusmarkdownv1.RecallPageResponse, error) {
	r.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), serveTimeLimit)
	defer cancel()
	return r.client.RecallPage(ctx, &corpusmarkdownv1.RecallPageRequest{Url: url})
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
	receiver := grpc.NewMarkdownRecallReceiver(recalledPages{}, "127.0.0.1:99999")

	if err := receiver.Serve(context.Background()); err == nil {
		t.Fatal("expected error when listen address cannot bind")
	}
}

func TestRecallPageAnswersWithTheMarkdownTheCorpusHolds(t *testing.T) {
	receiver := markdownRecallReceiverUnderTest(t, recalledPages{
		byRequestedURL: map[canonicalurl.CanonicalURL]markdownrecall.RecalledPage{
			canonicalurltest.CanonicalURLOf(t, recalledURL): {
				MarkdownURL:    canonicalurltest.CanonicalURLOf(t, recalledURL),
				StoredMarkdown: heldMarkdown,
			},
		},
	})

	response, err := receiver.RecallPage(recalledURL)
	if err != nil {
		t.Fatalf("recall page: %v", err)
	}
	if response.GetCanonicalUrl() != recalledURL {
		t.Errorf("canonicalUrl = %q, want %q", response.GetCanonicalUrl(), recalledURL)
	}
	if response.GetMarkdown() != "# Hi" {
		t.Errorf("markdown = %q, want %q", response.GetMarkdown(), "# Hi")
	}
	if !response.GetStoredAt().AsTime().Equal(storedAt) {
		t.Errorf("storedAt = %v, want %v", response.GetStoredAt().AsTime(), storedAt)
	}
	if response.GetVersion() != heldMarkdown.Version {
		t.Errorf("version = %q, want %q", response.GetVersion(), heldMarkdown.Version)
	}
}

func TestRecallPageAnswersWithTheURLTheMarkdownIsOf(t *testing.T) {
	const redirectedFrom = "http://example.com/"
	receiver := markdownRecallReceiverUnderTest(t, recalledPages{
		byRequestedURL: map[canonicalurl.CanonicalURL]markdownrecall.RecalledPage{
			canonicalurltest.CanonicalURLOf(t, redirectedFrom): {
				MarkdownURL:    canonicalurltest.CanonicalURLOf(t, recalledURL),
				StoredMarkdown: heldMarkdown,
			},
		},
	})

	response, err := receiver.RecallPage(redirectedFrom)
	if err != nil {
		t.Fatalf("recall page: %v", err)
	}
	if response.GetCanonicalUrl() != recalledURL {
		t.Errorf("canonicalUrl = %q, want %q", response.GetCanonicalUrl(), recalledURL)
	}
}

func TestRecallPageReportsNotFoundForAURLTheCorpusDoesNotHold(t *testing.T) {
	receiver := markdownRecallReceiverUnderTest(t, recalledPages{})

	_, err := receiver.RecallPage(recalledURL)

	if status.Code(err) != codes.NotFound {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.NotFound)
	}
}

func TestRecallPageRefusesARequestWhoseURLIsNotCanonicalizable(t *testing.T) {
	receiver := markdownRecallReceiverUnderTest(t, recalledPages{})

	_, err := receiver.RecallPage("")

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestRecallPageReportsACorpusFailureAsInternal(t *testing.T) {
	receiver := markdownRecallReceiverUnderTest(t, recalledPages{failure: errCorpusUnreachable})

	_, err := receiver.RecallPage(recalledURL)

	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want %v", status.Code(err), codes.Internal)
	}
}
