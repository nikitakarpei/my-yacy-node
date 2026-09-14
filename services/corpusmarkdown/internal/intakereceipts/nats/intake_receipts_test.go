package nats_test

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	intakereceiptsnats "github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/intakereceipts/nats"
	"github.com/nikitakarpei/yacy-rwi-node/natstestserver"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
)

const (
	pageURL      = "https://example.com/"
	otherPageURL = "https://example.com/other"
	corpus       = "corpusmarkdown"
	receiptWait  = 5 * time.Second
)

type recordingIntakeReceiptPublicationObserver struct {
	sentSubject          string
	encodingFailures     int
	publishingFailures   int
	confirmationFailures int
}

func (o *recordingIntakeReceiptPublicationObserver) IntakeReceiptSent(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	subject string,
) {
	o.sentSubject = subject
}

func (o *recordingIntakeReceiptPublicationObserver) IntakeReceiptEncodingFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ error,
) {
	o.encodingFailures++
}

func (o *recordingIntakeReceiptPublicationObserver) IntakeReceiptPublishingFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ string,
	_ error,
) {
	o.publishingFailures++
}

func (o *recordingIntakeReceiptPublicationObserver) IntakeReceiptConfirmationFailed(
	_ context.Context,
	_ canonicalurl.CanonicalURL,
	_ string,
	_ error,
) {
	o.confirmationFailures++
}

func TestKeptPageIsReportedOnTheSubjectOfThatPage(t *testing.T) {
	receipts, listener, observer := receiptsUnderTest(t, pagescrapecontract.KeptPageSubjectOf)
	page := canonicalurltest.CanonicalURLOf(t, pageURL)

	receipts.ReportKeptPage(context.Background(), page)

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
	if want := pagescrapecontract.KeptPageSubjectOf(page); observer.sentSubject != want {
		t.Errorf("observed the receipt on %q, want %q", observer.sentSubject, want)
	}
}

func TestRejectedPageIsReportedOnTheSubjectOfThatPage(t *testing.T) {
	receipts, listener, observer := receiptsUnderTest(
		t, pagescrapecontract.RejectedPageSubjectOf,
	)
	page := canonicalurltest.CanonicalURLOf(t, pageURL)

	receipts.ReportRejectedPage(context.Background(), page)

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
	if want := pagescrapecontract.RejectedPageSubjectOf(page); observer.sentSubject != want {
		t.Errorf("observed the receipt on %q, want %q", observer.sentSubject, want)
	}
}

func TestReceiptOfOnePageReachesNoListenerOfAnother(t *testing.T) {
	receipts, listener, _ := receiptsUnderTest(t, pagescrapecontract.KeptPageSubjectOf)

	receipts.ReportKeptPage(
		context.Background(),
		canonicalurltest.CanonicalURLOf(t, otherPageURL),
	)

	if _, err := listener.NextMsg(time.Second); err == nil {
		t.Error("a listener of one page heard the receipt of another")
	}
}

func TestReceiptThatIsNeverConfirmedIsObserved(t *testing.T) {
	receipts, _, observer := receiptsUnderTest(t, pagescrapecontract.KeptPageSubjectOf)
	ctx, stopWaiting := context.WithCancel(context.Background())
	stopWaiting()

	receipts.ReportKeptPage(ctx, canonicalurltest.CanonicalURLOf(t, pageURL))

	if observer.confirmationFailures != 1 {
		t.Errorf("observed %d unconfirmed receipts, want exactly one",
			observer.confirmationFailures)
	}
	if observer.sentSubject != "" {
		t.Errorf("observed a receipt sent on %q, want none", observer.sentSubject)
	}
}

func receiptsUnderTest(
	t *testing.T,
	subjectOf func(canonicalurl.CanonicalURL) string,
) (
	*intakereceiptsnats.IntakeReceipts,
	*nats.Subscription,
	*recordingIntakeReceiptPublicationObserver,
) {
	t.Helper()
	url := natstestserver.Start(t)
	listener := listenerOf(t, url, subjectOf(canonicalurltest.CanonicalURLOf(t, pageURL)))
	observer := &recordingIntakeReceiptPublicationObserver{}
	return intakereceiptsnats.NewIntakeReceipts(
		natstestserver.Connect(t, url), corpus, observer,
	), listener, observer
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
