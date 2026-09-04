package pageintake_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/corpusmarkdown/internal/pageintake"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/pullintaketest"
)

const (
	pageURL  = "https://example.org/a"
	pageHTML = "<html lang=\"en\"><head><title>Bicycles</title></head>" +
		"<body><p>Bicycles are ridden every day.</p></body></html>"
	pageHeading = "Bicycles"
)

type recordingCorpus struct {
	refuse bool
	mu     sync.Mutex
	stored map[string]string
}

func (c *recordingCorpus) Put(
	_ context.Context,
	canonicalURL canonicalurl.CanonicalURL,
	markdown []byte,
) error {
	if c.refuse {
		return errors.New("corpus write refused")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stored == nil {
		c.stored = map[string]string{}
	}
	c.stored[canonicalURL.String()] = string(markdown)
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
	mu            sync.Mutex
	offered       int
	stored        int
	noDocument    int
	noMarkdown    int
	storeFailures int
}

func (p *recordingProgress) PageOffered(context.Context, canonicalurl.CanonicalURL) {
	p.count(&p.offered)
}

func (p *recordingProgress) MarkdownStored(context.Context, canonicalurl.CanonicalURL) {
	p.count(&p.stored)
}

func (p *recordingProgress) NoDocumentExtracted(
	context.Context,
	canonicalurl.CanonicalURL,
	error,
) {
	p.count(&p.noDocument)
}

func (p *recordingProgress) NoMarkdownDerived(context.Context, canonicalurl.CanonicalURL) {
	p.count(&p.noMarkdown)
}

func (p *recordingProgress) MarkdownNotStored(
	context.Context,
	canonicalurl.CanonicalURL,
	error,
) {
	p.count(&p.storeFailures)
}

func (p *recordingProgress) count(counter *int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	*counter++
}

type pageIntake struct {
	message  *pullintaketest.Message
	corpus   *recordingCorpus
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
	return runPageIntakeInto(t, message, &recordingCorpus{}, &recordingIntakeReceipts{})
}

func runPageIntakeInto(
	t *testing.T,
	message *pullintaketest.Message,
	corpus *recordingCorpus,
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
		Corpus:                     corpus,
		IntakeReceipts:             receipts,
		PageIntakeObserver:         progress,
		PageOfferIntakeConcurrency: 1,
	})
	if err := consumer.Run(context.Background()); err != nil {
		t.Fatalf("run intake: %v", err)
	}
	return pageIntake{message: message, corpus: corpus, receipts: receipts, progress: progress}
}

func TestOfferedPageIsStoredAsMarkdownAndReportedAsKept(t *testing.T) {
	intake := runPageIntake(t, offeredPage(t, pageHTML))

	markdown, stored := intake.corpus.stored[pageURL]
	if !stored {
		t.Fatalf("stored %v, want the markdown of %s", intake.corpus.stored, pageURL)
	}
	if !strings.Contains(markdown, pageHeading) {
		t.Errorf("stored markdown = %q, want it to carry %q", markdown, pageHeading)
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

	if len(intake.corpus.stored) != 0 {
		t.Errorf("stored %v, want nothing", intake.corpus.stored)
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

func TestOfferComesBackWhenTheCorpusWriteFails(t *testing.T) {
	intake := runPageIntakeInto(
		t, offeredPage(t, pageHTML), &recordingCorpus{refuse: true}, &recordingIntakeReceipts{},
	)

	if len(intake.receipts.kept)+len(intake.receipts.rejected) != 0 {
		t.Errorf("sent a receipt for a page it did not dispose of")
	}
	if intake.progress.storeFailures != 1 {
		t.Errorf("observed %d corpus write failures, want exactly one",
			intake.progress.storeFailures)
	}
	if settlement := intake.message.Settlement(t); settlement != pullintaketest.HeldBack {
		t.Errorf("the offer was %s, want it %s", settlement, pullintaketest.HeldBack)
	}
}

func TestOfferedPageThatCannotBeReadHaltsIntake(t *testing.T) {
	message := &pullintaketest.Message{Body: []byte("not json")}
	consumer := pageintake.NewOfferedPageConsumer(pageintake.Config{
		Source:                     pullintaketest.MessageSourceOf(message),
		Corpus:                     &recordingCorpus{},
		IntakeReceipts:             &recordingIntakeReceipts{},
		PageIntakeObserver:         &recordingProgress{},
		PageOfferIntakeConcurrency: 1,
	})

	err := consumer.Run(context.Background())
	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Fatalf("intake returned %v, want it to halt on an offer it cannot read", err)
	}
}
