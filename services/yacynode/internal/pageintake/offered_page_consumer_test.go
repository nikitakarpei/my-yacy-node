package pageintake_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/pullintaketest"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagerwi"
)

const (
	offeredPageURL = "https://example.com/"
	landedPageURL  = "http://archive.example/replay/https://example.com/"
	pageTitle      = "Hi"
)

type recordingPages struct {
	receipt  pageadmission.Receipt
	err      error
	received []pagerwi.PageRWI
}

func (r *recordingPages) Receive(
	_ context.Context,
	page pagerwi.PageRWI,
) (pageadmission.Receipt, error) {
	r.received = append(r.received, page)

	return r.receipt, r.err
}

type recordingIntakeReceipts struct {
	kept     []canonicalurl.CanonicalURL
	rejected []canonicalurl.CanonicalURL
}

func (r *recordingIntakeReceipts) ReportKeptPage(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	r.kept = append(r.kept, pageURL)
}

func (r *recordingIntakeReceipts) ReportRejectedPage(
	_ context.Context,
	pageURL canonicalurl.CanonicalURL,
) {
	r.rejected = append(r.rejected, pageURL)
}

type recordingPageIntakeObserver struct {
	disposals           []string
	pagesOffered        int
	pagesAdmitted       int
	postingsAdmitted    int
	postingsNotAdmitted []int
	admissionFailures   []error
}

func (r *recordingPageIntakeObserver) OfferedPageInvalid(context.Context) {
	r.disposals = append(r.disposals, "invalid_message")
}

func (r *recordingPageIntakeObserver) PageOffered(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	r.pagesOffered++
}

func (r *recordingPageIntakeObserver) DocumentExtractionFailed(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	error,
) {
	r.disposals = append(r.disposals, "document_extraction_failed")
}

func (r *recordingPageIntakeObserver) NoIndexDerived(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	r.disposals = append(r.disposals, "no_index_derived")
}

func (r *recordingPageIntakeObserver) PageAdmitted(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
	amountOfPostings int,
) {
	r.pagesAdmitted++
	r.postingsAdmitted += amountOfPostings
}

func (r *recordingPageIntakeObserver) PageAdmissionBusy(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
	amountOfPostings int,
) {
	r.disposals = append(r.disposals, "page_admission_busy")
	r.postingsNotAdmitted = append(r.postingsNotAdmitted, amountOfPostings)
}

func (r *recordingPageIntakeObserver) PageAdmissionFailed(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
	amountOfPostings int,
	cause error,
) {
	r.disposals = append(r.disposals, "page_admission_failed")
	r.postingsNotAdmitted = append(r.postingsNotAdmitted, amountOfPostings)
	r.admissionFailures = append(r.admissionFailures, cause)
}

func (r *recordingPageIntakeObserver) PageIndexed(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	r.disposals = append(r.disposals, "indexed")
}

func offeredPageMessage(t *testing.T, text string) *pullintaketest.Message {
	t.Helper()

	return offeredPage(t, pagescrapecontract.OfferedPage{
		PageURL:     canonicalurltest.CanonicalURLOf(t, offeredPageURL),
		LandedURL:   canonicalurltest.CanonicalURLOf(t, offeredPageURL),
		ContentType: "text/html",
		Body: []byte(
			`<html lang="en"><head><title>` + pageTitle + `</title></head>` +
				`<body><p>` + text + `</p></body></html>`,
		),
	})
}

func offeredPage(
	t *testing.T,
	page pagescrapecontract.OfferedPage,
) *pullintaketest.Message {
	t.Helper()

	data, err := pagescrapecontract.MarshalOfferedPage(page)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	return &pullintaketest.Message{Body: data}
}

type intakeCollaborators struct {
	pageReceiver       pageadmission.PageReceiver
	intakeReceipts     pageintake.IntakeReceipts
	pageIntakeObserver pageintake.PageIntakeObserver
}

func run(t *testing.T, msg jetstream.Msg, pages pageadmission.PageReceiver) error {
	return runWith(t, msg, intakeCollaborators{
		pageReceiver:       pages,
		intakeReceipts:     &recordingIntakeReceipts{},
		pageIntakeObserver: pageintake.PageIntakeObservers{},
	})
}

func runWith(
	t *testing.T,
	msg jetstream.Msg,
	collaborators intakeCollaborators,
) error {
	t.Helper()

	formatDerivations, err := pageformats.New()
	if err != nil {
		t.Fatalf("page formats: %v", err)
	}

	return pageintake.NewOfferedPageConsumer(
		pageintake.OfferedPageConsumerConfig{
			OfferedPageSource:          pullintaketest.MessageSourceOf(msg),
			FormatDerivations:          formatDerivations,
			PageReceiver:               collaborators.pageReceiver,
			IntakeReceipts:             collaborators.intakeReceipts,
			PageIntakeObserver:         collaborators.pageIntakeObserver,
			PageOfferIntakeConcurrency: 1,
		}).Run(context.Background())
}

func TestOfferedPageIsIndexedAndReportedAsKept(t *testing.T) {
	observer := &recordingPageIntakeObserver{}
	receipts := &recordingIntakeReceipts{}
	pages := &recordingPages{}
	message := offeredPageMessage(t, "alpha beta")

	if err := runWith(t, message, intakeCollaborators{
		pageReceiver:       pages,
		intakeReceipts:     receipts,
		pageIntakeObserver: observer,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if got := observer.disposals; len(got) != 1 || got[0] != "indexed" {
		t.Errorf("disposals = %v, want indexed", got)
	}
	if observer.pagesOffered != 1 || observer.pagesAdmitted != 1 {
		t.Errorf("pages offered = %d and admitted = %d, want one of each",
			observer.pagesOffered, observer.pagesAdmitted)
	}
	if len(pages.received) != 1 {
		t.Fatalf("admitted %d pages, want one for the offered page", len(pages.received))
	}
	admitted := pages.received[0]
	if admitted.Metadata.Address != offeredPageURL {
		t.Errorf("admitted address = %q, want the offered page", admitted.Metadata.Address)
	}
	if admitted.Metadata.Title != pageTitle {
		t.Errorf("admitted title = %q, want the extracted title", admitted.Metadata.Title)
	}
	if len(receipts.kept) != 1 || receipts.kept[0].String() != offeredPageURL {
		t.Errorf("kept receipts = %v, want one for the offered page", receipts.kept)
	}
	assertWordsAdmitted(t, pages, "alpha", "beta")
}

func TestPageThatLandedElsewhereIsIndexedUnderTheOfferedURL(t *testing.T) {
	pages := &recordingPages{}

	message := offeredPage(t, pagescrapecontract.OfferedPage{
		PageURL:     canonicalurltest.CanonicalURLOf(t, offeredPageURL),
		LandedURL:   canonicalurltest.CanonicalURLOf(t, landedPageURL),
		ContentType: "text/html",
		Body:        []byte(`<html lang="en"><body><p>alpha</p></body></html>`),
	})
	if err := run(t, message, pages); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if len(pages.received) != 1 ||
		pages.received[0].Metadata.Address != offeredPageURL {
		t.Fatalf("admitted %+v, want one page under the offered url", pages.received)
	}
}

func TestConsumerAdmitsAPageInOneCall(t *testing.T) {
	pages := &recordingPages{}

	if err := run(
		t,
		offeredPageMessage(t, "alpha beta gamma delta epsilon"),
		pages,
	); err != nil {
		t.Fatalf("run: %v", err)
	}

	assertWordsAdmitted(t, pages, "alpha", "beta", "gamma", "delta", "epsilon")
}

func TestPageNoDocumentIsExtractedFromIsReportedAsRejected(t *testing.T) {
	observer := &recordingPageIntakeObserver{}
	receipts := &recordingIntakeReceipts{}
	pages := &recordingPages{}

	message := offeredPage(t, pagescrapecontract.OfferedPage{
		PageURL:     canonicalurltest.CanonicalURLOf(t, offeredPageURL),
		LandedURL:   canonicalurltest.CanonicalURLOf(t, offeredPageURL),
		ContentType: "application/pdf",
		Body:        []byte("%PDF-1.4"),
	})
	if err := runWith(t, message, intakeCollaborators{
		pageReceiver:       pages,
		intakeReceipts:     receipts,
		pageIntakeObserver: observer,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if len(pages.received) != 0 {
		t.Errorf("admitted %v, want nothing", pages.received)
	}
	if len(receipts.rejected) != 1 {
		t.Errorf("rejected receipts = %v, want one for the offered page", receipts.rejected)
	}
	if got := observer.disposals; len(got) != 1 || got[0] != "document_extraction_failed" {
		t.Errorf("disposals = %v, want document_extraction_failed", got)
	}
}

type admissionDisposalExpectation struct {
	pages            *recordingPages
	wantDisposal     string
	wantFailureCause bool
}

func TestConsumerReportsWhyAdmissionReturnedTheOfferedPage(t *testing.T) {
	cause := errors.New("admission failed")
	for name, expectation := range map[string]admissionDisposalExpectation{
		"busy": {
			pages:        &recordingPages{receipt: pageadmission.Receipt{Busy: true}},
			wantDisposal: "page_admission_busy",
		},
		"failed": {
			pages:            &recordingPages{err: cause},
			wantDisposal:     "page_admission_failed",
			wantFailureCause: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			observer := &recordingPageIntakeObserver{}
			message := offeredPageMessage(t, "alpha")

			if err := runWith(t, message, intakeCollaborators{
				pageReceiver:       expectation.pages,
				intakeReceipts:     &recordingIntakeReceipts{},
				pageIntakeObserver: observer,
			}); err != nil {
				t.Fatalf("run: %v", err)
			}

			if action := message.Settlement(t); action != pullintaketest.HeldBack {
				t.Errorf("action = %q, want nak", action)
			}
			assertAdmissionDisposal(t, observer, expectation)
		})
	}
}

func assertAdmissionDisposal(
	t *testing.T,
	observer *recordingPageIntakeObserver,
	expectation admissionDisposalExpectation,
) {
	t.Helper()

	if got := observer.disposals; len(got) != 1 || got[0] != expectation.wantDisposal {
		t.Errorf("disposals = %v, want %s", got, expectation.wantDisposal)
	}
	if observer.pagesAdmitted != 0 || observer.postingsAdmitted != 0 {
		t.Errorf("pages admitted = %d and postings admitted = %d, want none",
			observer.pagesAdmitted, observer.postingsAdmitted)
	}
	admitted := expectation.pages.received
	if len(admitted) != 1 {
		t.Fatalf("admission calls = %d, want one", len(admitted))
	}
	if len(observer.postingsNotAdmitted) != 1 ||
		observer.postingsNotAdmitted[0] != len(admitted[0].Postings) {
		t.Errorf("postings not admitted = %v, want %d",
			observer.postingsNotAdmitted, len(admitted[0].Postings))
	}
	if got := len(observer.admissionFailures) != 0; got != expectation.wantFailureCause {
		t.Errorf("failure cause reported = %t, want %t", got, expectation.wantFailureCause)
	}
}

func TestConsumerHaltsOnAnUndecodableMessage(t *testing.T) {
	msg := &pullintaketest.Message{Body: []byte("not an offered page")}

	err := run(t, msg, &recordingPages{})

	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Fatalf("err = %v, want a poison message halt", err)
	}
	if settled := msg.Settlements(); len(settled) != 0 {
		t.Errorf("undecodable message settled %v, want it left pending", settled)
	}
}

func assertWordsAdmitted(t *testing.T, pages *recordingPages, words ...string) {
	t.Helper()

	if len(pages.received) != 1 {
		t.Fatalf("admitted over %d calls, want one call for the page", len(pages.received))
	}
	admitted := map[yacymodel.Hash]bool{}
	for _, posting := range pages.received[0].Postings {
		admitted[posting.WordHash] = true
	}
	for _, word := range words {
		if !admitted[yacymodel.WordHash(word)] {
			t.Errorf("word %q should be admitted, got %v", word, pages.received[0].Postings)
		}
	}
}
