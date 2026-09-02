package nats_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	intakereceiptsnats "github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/intakereceipts/nats"
)

const (
	pageURL      = "https://example.com/"
	otherPageURL = "https://example.com/other"
	corpus       = "yacynode"
	receiptWait  = 5 * time.Second
)

func TestKeptPageIsReportedOnTheSubjectOfThatPage(t *testing.T) {
	receipts, listener := receiptsUnderTest(t, pagescrapecontract.KeptPageSubjectOf)
	page := canonicalurltest.CanonicalURLOf(t, pageURL)

	if err := receipts.ReportKeptPage(context.Background(), page); err != nil {
		t.Fatalf("report the kept page: %v", err)
	}

	kept, err := pagescrapecontract.UnmarshalKeptPage(heardReceipt(t, listener))
	if err != nil {
		t.Fatalf("unmarshal the kept page: %v", err)
	}
	if kept.PageURL != page {
		t.Errorf("reported url = %q, want %q", kept.PageURL, page)
	}
	if kept.Corpus != corpus {
		t.Errorf("reported corpus = %q, want %q", kept.Corpus, corpus)
	}
}

func TestRejectedPageIsReportedOnTheSubjectOfThatPage(t *testing.T) {
	receipts, listener := receiptsUnderTest(t, pagescrapecontract.RejectedPageSubjectOf)
	page := canonicalurltest.CanonicalURLOf(t, pageURL)

	if err := receipts.ReportRejectedPage(context.Background(), page); err != nil {
		t.Fatalf("report the rejected page: %v", err)
	}

	rejected, err := pagescrapecontract.UnmarshalRejectedPage(heardReceipt(t, listener))
	if err != nil {
		t.Fatalf("unmarshal the rejected page: %v", err)
	}
	if rejected.PageURL != page {
		t.Errorf("reported url = %q, want %q", rejected.PageURL, page)
	}
	if rejected.Corpus != corpus {
		t.Errorf("reported corpus = %q, want %q", rejected.Corpus, corpus)
	}
}

func TestReceiptOfOnePageReachesNoListenerOfAnother(t *testing.T) {
	receipts, listener := receiptsUnderTest(t, pagescrapecontract.KeptPageSubjectOf)

	err := receipts.ReportKeptPage(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, otherPageURL),
	)
	if err != nil {
		t.Fatalf("report the kept page: %v", err)
	}

	if _, err := listener.NextMsg(time.Second); err == nil {
		t.Error("a listener of one page heard the receipt of another")
	}
}

func TestReceiptsStopWhenTheCallerStopsWaiting(t *testing.T) {
	receipts, _ := receiptsUnderTest(t, pagescrapecontract.KeptPageSubjectOf)
	ctx, stopWaiting := context.WithCancel(context.Background())
	stopWaiting()

	err := receipts.ReportKeptPage(ctx, canonicalurltest.CanonicalURLOf(t, pageURL))

	if !errors.Is(err, context.Canceled) {
		t.Errorf("report the kept page = %v, want the cancellation", err)
	}
}

func receiptsUnderTest(
	t *testing.T,
	subjectOf func(canonicalurl.CanonicalURL) string,
) (*intakereceiptsnats.IntakeReceipts, *nats.Subscription) {
	t.Helper()
	url := natstestserver.Start(t)
	listener := listenerOf(t, url, subjectOf(canonicalurltest.CanonicalURLOf(t, pageURL)))
	return intakereceiptsnats.NewIntakeReceipts(
		natstestserver.Connect(t, url), corpus,
	), listener
}

func listenerOf(t *testing.T, url string, subject string) *nats.Subscription {
	t.Helper()
	connection := natstestserver.Connect(t, url)
	listener, err := connection.SubscribeSync(subject)
	if err != nil {
		t.Fatalf("listen for intake receipts: %v", err)
	}
	if err := connection.Flush(); err != nil {
		t.Fatalf("flush the listener: %v", err)
	}
	return listener
}

func heardReceipt(t *testing.T, listener *nats.Subscription) []byte {
	t.Helper()
	message, err := listener.NextMsg(receiptWait)
	if err != nil {
		t.Fatalf("wait for the receipt: %v", err)
	}
	return message.Data
}
