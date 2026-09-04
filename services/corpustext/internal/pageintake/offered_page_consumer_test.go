package pageintake_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/corpustext/internal/pageintake"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/searchdocument"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/pullintaketest"
)

const (
	pageURL  = "https://example.org/a"
	pageHTML = "<html lang=\"en\"><head><title>Bicycles</title></head>" +
		"<body><p>Bicycles are ridden every day.</p></body></html>"
)

type recordingIndex struct {
	refuse    bool
	mu        sync.Mutex
	documents []searchdocument.Document
}

func (i *recordingIndex) Index(_ context.Context, document searchdocument.Document) error {
	if i.refuse {
		return errors.New("index refused")
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.documents = append(i.documents, document)
	return nil
}

type recordingIntakeReceipts struct {
	mu       sync.Mutex
	kept     []string
	rejected []string
}

func (r *recordingIntakeReceipts) ReportKeptPage(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.kept = append(r.kept, pageURL.String())
}

func (r *recordingIntakeReceipts) ReportRejectedPage(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rejected = append(r.rejected, pageURL.String())
}

type recordingProgress struct {
	mu             sync.Mutex
	offered        int
	indexed        int
	noDocument     int
	noReadableText int
	indexFailures  int
	observations   int
}

func (p *recordingProgress) PageOffered(context.Context, canonicalurl.CanonicalURL) {
	p.count(&p.offered)
}

func (p *recordingProgress) PageIndexed(context.Context, canonicalurl.CanonicalURL) {
	p.count(&p.indexed)
}

func (p *recordingProgress) NoDocumentExtracted(
	context.Context,
	canonicalurl.CanonicalURL,
	error,
) {
	p.count(&p.noDocument)
}

func (p *recordingProgress) NoReadableTextDerived(context.Context, canonicalurl.CanonicalURL) {
	p.count(&p.noReadableText)
}

func (p *recordingProgress) IndexFailed(context.Context, canonicalurl.CanonicalURL, error) {
	p.count(&p.indexFailures)
}

func (p *recordingProgress) IndexObserved(context.Context, time.Duration) {
	p.count(&p.observations)
}

func (p *recordingProgress) count(counter *int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	*counter++
}

type pageIntake struct {
	message  *pullintaketest.Message
	index    *recordingIndex
	receipts *recordingIntakeReceipts
	progress *recordingProgress
}

func offeredPage(t *testing.T, body string) *pullintaketest.Message {
	t.Helper()
	return offeredPageOfType(t, "text/html", body)
}

func offeredPageOfType(t *testing.T, contentType, body string) *pullintaketest.Message {
	t.Helper()
	pageCanonicalURL := canonicalurltest.CanonicalURLOf(t, pageURL)
	data, err := pagescrapecontract.MarshalOfferedPage(pagescrapecontract.OfferedPage{
		PageURL:     pageCanonicalURL,
		LandedURL:   pageCanonicalURL,
		ContentType: contentType,
		Body:        []byte(body),
	})
	if err != nil {
		t.Fatalf("marshal offered page: %v", err)
	}
	return &pullintaketest.Message{Body: data}
}

func runPageIntake(t *testing.T, message *pullintaketest.Message) pageIntake {
	t.Helper()
	return runPageIntakeInto(t, message, &recordingIndex{}, &recordingIntakeReceipts{})
}

func runPageIntakeInto(
	t *testing.T,
	message *pullintaketest.Message,
	index *recordingIndex,
	receipts *recordingIntakeReceipts,
) pageIntake {
	t.Helper()
	formatDerivations, err := pageformats.New()
	if err != nil {
		t.Fatalf("open the format derivations: %v", err)
	}
	progress := &recordingProgress{}
	consumer := pageintake.NewOfferedPageConsumer(pageintake.Config{
		Source:                     pullintaketest.MessageSourceOf(message),
		FormatDerivations:          formatDerivations,
		SearchIndex:                index,
		IntakeReceipts:             receipts,
		IntakeProgress:             progress,
		PageOfferIntakeConcurrency: 1,
	})
	if err := consumer.Run(context.Background()); err != nil {
		t.Fatalf("run intake: %v", err)
	}
	return pageIntake{message: message, index: index, receipts: receipts, progress: progress}
}

func TestOfferedPageIsIndexedAndReportedAsKept(t *testing.T) {
	intake := runPageIntake(t, offeredPage(t, pageHTML))

	if len(intake.index.documents) != 1 {
		t.Fatalf("indexed %d documents, want exactly one", len(intake.index.documents))
	}
	if got := intake.index.documents[0].URL; got != pageURL {
		t.Errorf("indexed %s, want %s", got, pageURL)
	}
	if len(intake.receipts.kept) != 1 || intake.receipts.kept[0] != pageURL {
		t.Errorf("reported %v as kept, want only %s", intake.receipts.kept, pageURL)
	}
	if settlement := intake.message.Settlement(t); settlement != pullintaketest.Acknowledged {
		t.Errorf("the offer was %s, want it %s", settlement, pullintaketest.Acknowledged)
	}
}

func TestPageNoDocumentIsExtractedFromIsReportedAsRejected(t *testing.T) {
	intake := runPageIntake(t, offeredPageOfType(t, "application/octet-stream", "\x00\x01"))

	if len(intake.index.documents) != 0 {
		t.Errorf("indexed %d documents, want none", len(intake.index.documents))
	}
	if len(intake.receipts.rejected) != 1 {
		t.Fatalf("reported %d pages as rejected, want exactly one",
			len(intake.receipts.rejected))
	}
	if intake.progress.noDocument != 1 {
		t.Errorf("observed %d pages no document came out of, want exactly one",
			intake.progress.noDocument)
	}
	if settlement := intake.message.Settlement(t); settlement != pullintaketest.Acknowledged {
		t.Errorf("the offer was %s, want it %s", settlement, pullintaketest.Acknowledged)
	}
}

func TestOfferComesBackWhenTheIndexFails(t *testing.T) {
	intake := runPageIntakeInto(
		t, offeredPage(t, pageHTML), &recordingIndex{refuse: true}, &recordingIntakeReceipts{},
	)

	if len(intake.receipts.kept)+len(intake.receipts.rejected) != 0 {
		t.Errorf("sent a receipt for a page it did not dispose of")
	}
	if settlement := intake.message.Settlement(t); settlement != pullintaketest.HeldBack {
		t.Errorf("the offer was %s, want it %s", settlement, pullintaketest.HeldBack)
	}
}

func TestOfferedPageThatCannotBeReadHaltsIntake(t *testing.T) {
	message := &pullintaketest.Message{Body: []byte("not json")}
	consumer := pageintake.NewOfferedPageConsumer(pageintake.Config{
		Source:                     pullintaketest.MessageSourceOf(message),
		SearchIndex:                &recordingIndex{},
		IntakeReceipts:             &recordingIntakeReceipts{},
		IntakeProgress:             &recordingProgress{},
		PageOfferIntakeConcurrency: 1,
	})

	err := consumer.Run(context.Background())
	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Fatalf("intake returned %v, want it to halt on an offer it cannot read", err)
	}
}
