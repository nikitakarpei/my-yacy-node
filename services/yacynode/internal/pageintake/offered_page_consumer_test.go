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
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageintake"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
)

const (
	offeredPageURL = "https://example.com/"
	landedPageURL  = "http://archive.example/replay/https://example.com/"
	pageTitle      = "Hi"
)

type recordingURLs struct {
	receipt  urlmeta.Receipt
	err      error
	received []yacymodel.URLMetadata
}

func (r *recordingURLs) Receive(
	_ context.Context,
	metadata []yacymodel.URLMetadata,
) (urlmeta.Receipt, error) {
	r.received = append(r.received, metadata...)

	return r.receipt, r.err
}

type recordingPostings struct {
	receipt rwiadmission.Receipt
	err     error
	calls   [][]yacymodel.RWIPosting
}

func (r *recordingPostings) Receive(
	_ context.Context,
	postings []yacymodel.RWIPosting,
) (rwiadmission.Receipt, error) {
	r.calls = append(r.calls, postings)

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
	urlMetadataAdmitted int
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

func (r *recordingPageIntakeObserver) URLMetadataAdmitted(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	r.urlMetadataAdmitted++
}

func (r *recordingPageIntakeObserver) URLMetadataAdmissionBusy(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	r.disposals = append(r.disposals, "url_metadata_admission_busy")
}

func (r *recordingPageIntakeObserver) URLMetadataAdmissionFailed(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
	cause error,
) {
	r.disposals = append(r.disposals, "url_metadata_admission_failed")
	r.admissionFailures = append(r.admissionFailures, cause)
}

func (r *recordingPageIntakeObserver) PostingsAdmitted(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
	postings int,
) {
	r.postingsAdmitted += postings
}

func (r *recordingPageIntakeObserver) PostingsAdmissionBusy(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
	postings int,
) {
	r.disposals = append(r.disposals, "postings_admission_busy")
	r.postingsNotAdmitted = append(r.postingsNotAdmitted, postings)
}

func (r *recordingPageIntakeObserver) PostingsAdmissionFailed(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
	postings int,
	cause error,
) {
	r.disposals = append(r.disposals, "postings_admission_failed")
	r.postingsNotAdmitted = append(r.postingsNotAdmitted, postings)
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
	urlReceiver        urlmeta.URLReceiver
	postingReceiver    rwiadmission.PostingReceiver
	intakeReceipts     pageintake.IntakeReceipts
	pageIntakeObserver pageintake.PageIntakeObserver
}

func run(
	t *testing.T,
	msg jetstream.Msg,
	urls urlmeta.URLReceiver,
	postings rwiadmission.PostingReceiver,
) error {
	return runWith(t, msg, intakeCollaborators{
		urlReceiver:        urls,
		postingReceiver:    postings,
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
			URLReceiver:                collaborators.urlReceiver,
			PostingReceiver:            collaborators.postingReceiver,
			IntakeReceipts:             collaborators.intakeReceipts,
			PageIntakeObserver:         collaborators.pageIntakeObserver,
			PageOfferIntakeConcurrency: 1,
		}).Run(context.Background())
}

func TestOfferedPageIsIndexedAndReportedAsKept(t *testing.T) {
	progress := &recordingPageIntakeObserver{}
	receipts := &recordingIntakeReceipts{}
	urls := &recordingURLs{}
	postings := &recordingPostings{}
	message := offeredPageMessage(t, "alpha beta")

	if err := runWith(t, message, intakeCollaborators{
		urlReceiver:        urls,
		postingReceiver:    postings,
		intakeReceipts:     receipts,
		pageIntakeObserver: progress,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if got := progress.disposals; len(got) != 1 || got[0] != "indexed" {
		t.Errorf("disposals = %v, want indexed", got)
	}
	if progress.pagesOffered != 1 {
		t.Errorf("pages offered = %d, want 1", progress.pagesOffered)
	}
	if len(urls.received) != 1 || urls.received[0].Address != offeredPageURL {
		t.Fatalf("stored metadata %+v, want one row for the offered page", urls.received)
	}
	if urls.received[0].Title != pageTitle {
		t.Errorf("stored title = %q, want the extracted title", urls.received[0].Title)
	}
	if len(receipts.kept) != 1 || receipts.kept[0].String() != offeredPageURL {
		t.Errorf("kept receipts = %v, want one for the offered page", receipts.kept)
	}
	assertWordsAdmitted(t, postings, "alpha", "beta")
}

func TestPageThatLandedElsewhereIsIndexedUnderTheOfferedURL(t *testing.T) {
	urls := &recordingURLs{}

	message := offeredPage(t, pagescrapecontract.OfferedPage{
		PageURL:     canonicalurltest.CanonicalURLOf(t, offeredPageURL),
		LandedURL:   canonicalurltest.CanonicalURLOf(t, landedPageURL),
		ContentType: "text/html",
		Body:        []byte(`<html lang="en"><body><p>alpha</p></body></html>`),
	})
	if err := run(t, message, urls, &recordingPostings{}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if len(urls.received) != 1 || urls.received[0].Address != offeredPageURL {
		t.Fatalf("stored metadata %+v, want one row under the offered url", urls.received)
	}
}

func TestConsumerAdmitsEveryPostingOfAPageInOneCall(t *testing.T) {
	postings := &recordingPostings{}

	if err := run(t, offeredPageMessage(t, "alpha beta gamma delta epsilon"),
		&recordingURLs{}, postings); err != nil {
		t.Fatalf("run: %v", err)
	}

	assertWordsAdmitted(t, postings, "alpha", "beta", "gamma", "delta", "epsilon")
}

func TestPageNoDocumentIsExtractedFromIsReportedAsRejected(t *testing.T) {
	progress := &recordingPageIntakeObserver{}
	receipts := &recordingIntakeReceipts{}
	postings := &recordingPostings{}

	message := offeredPage(t, pagescrapecontract.OfferedPage{
		PageURL:     canonicalurltest.CanonicalURLOf(t, offeredPageURL),
		LandedURL:   canonicalurltest.CanonicalURLOf(t, offeredPageURL),
		ContentType: "application/pdf",
		Body:        []byte("%PDF-1.4"),
	})
	if err := runWith(t, message, intakeCollaborators{
		urlReceiver:        &recordingURLs{},
		postingReceiver:    postings,
		intakeReceipts:     receipts,
		pageIntakeObserver: progress,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if len(postings.calls) != 0 {
		t.Errorf("stored %v, want nothing", postings.calls)
	}
	if len(receipts.rejected) != 1 {
		t.Errorf("rejected receipts = %v, want one for the offered page", receipts.rejected)
	}
	if got := progress.disposals; len(got) != 1 || got[0] != "document_extraction_failed" {
		t.Errorf("disposals = %v, want document_extraction_failed", got)
	}
}

type admissionDisposalExpectation struct {
	urls             *recordingURLs
	postings         *recordingPostings
	wantDisposal     string
	wantURLMetadata  int
	wantPostingCalls int
	wantFailureCause bool
}

func TestConsumerReportsWhichAdmissionReturnedTheOfferedPage(t *testing.T) {
	cause := errors.New("admission failed")
	for name, expectation := range map[string]admissionDisposalExpectation{
		"url busy": {
			urls:         &recordingURLs{receipt: urlmeta.Receipt{Busy: true}},
			postings:     &recordingPostings{},
			wantDisposal: "url_metadata_admission_busy",
		},
		"url failed": {
			urls:             &recordingURLs{err: cause},
			postings:         &recordingPostings{},
			wantDisposal:     "url_metadata_admission_failed",
			wantFailureCause: true,
		},
		"url rejected": {
			urls: &recordingURLs{receipt: urlmeta.Receipt{
				ErrorURL: []yacymodel.URLHash{{}},
			}},
			postings:         &recordingPostings{},
			wantDisposal:     "url_metadata_admission_failed",
			wantFailureCause: true,
		},
		"posting busy": {
			urls:             &recordingURLs{},
			postings:         &recordingPostings{receipt: rwiadmission.Receipt{Busy: true}},
			wantDisposal:     "postings_admission_busy",
			wantURLMetadata:  1,
			wantPostingCalls: 1,
		},
		"posting failed": {
			urls:             &recordingURLs{},
			postings:         &recordingPostings{err: cause},
			wantDisposal:     "postings_admission_failed",
			wantURLMetadata:  1,
			wantPostingCalls: 1,
			wantFailureCause: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			progress := &recordingPageIntakeObserver{}
			message := offeredPageMessage(t, "alpha")

			if err := runWith(t, message, intakeCollaborators{
				urlReceiver:        expectation.urls,
				postingReceiver:    expectation.postings,
				intakeReceipts:     &recordingIntakeReceipts{},
				pageIntakeObserver: progress,
			}); err != nil {
				t.Fatalf("run: %v", err)
			}

			if action := message.Settlement(t); action != pullintaketest.HeldBack {
				t.Errorf("action = %q, want nak", action)
			}
			assertAdmissionDisposal(t, progress, expectation)
		})
	}
}

func assertAdmissionDisposal(
	t *testing.T,
	progress *recordingPageIntakeObserver,
	expectation admissionDisposalExpectation,
) {
	t.Helper()

	if got := progress.disposals; len(got) != 1 || got[0] != expectation.wantDisposal {
		t.Errorf("disposals = %v, want %s", got, expectation.wantDisposal)
	}
	if progress.urlMetadataAdmitted != expectation.wantURLMetadata {
		t.Errorf("url metadata admitted = %d, want %d",
			progress.urlMetadataAdmitted, expectation.wantURLMetadata)
	}
	if progress.postingsAdmitted != 0 {
		t.Errorf("postings admitted = %d, want 0", progress.postingsAdmitted)
	}
	if len(expectation.postings.calls) != expectation.wantPostingCalls {
		t.Errorf("posting admission calls = %d, want %d",
			len(expectation.postings.calls), expectation.wantPostingCalls)
	}
	if expectation.wantPostingCalls == 0 && len(progress.postingsNotAdmitted) != 0 {
		t.Errorf("postings not admitted = %v, want none", progress.postingsNotAdmitted)
	}
	if expectation.wantPostingCalls == 1 &&
		(len(progress.postingsNotAdmitted) != 1 ||
			progress.postingsNotAdmitted[0] != len(expectation.postings.calls[0])) {
		t.Errorf("postings not admitted = %v, want %d",
			progress.postingsNotAdmitted, len(expectation.postings.calls[0]))
	}
	if got := len(progress.admissionFailures) != 0; got != expectation.wantFailureCause {
		t.Errorf("failure cause reported = %t, want %t", got, expectation.wantFailureCause)
	}
}

func TestConsumerHaltsOnAnUndecodableMessage(t *testing.T) {
	msg := &pullintaketest.Message{Body: []byte("not an offered page")}

	err := run(t, msg, &recordingURLs{}, &recordingPostings{})

	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Fatalf("err = %v, want a poison message halt", err)
	}
	if settled := msg.Settlements(); len(settled) != 0 {
		t.Errorf("undecodable message settled %v, want it left pending", settled)
	}
}

func assertWordsAdmitted(t *testing.T, postings *recordingPostings, words ...string) {
	t.Helper()

	if len(postings.calls) != 1 {
		t.Fatalf("admitted over %d calls, want one call for the page", len(postings.calls))
	}
	admitted := map[yacymodel.Hash]bool{}
	for _, posting := range postings.calls[0] {
		admitted[posting.WordHash] = true
	}
	for _, word := range words {
		if !admitted[yacymodel.WordHash(word)] {
			t.Errorf("word %q should be admitted, got %v", word, postings.calls[0])
		}
	}
}
