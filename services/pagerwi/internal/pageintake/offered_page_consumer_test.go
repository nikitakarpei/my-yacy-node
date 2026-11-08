package pageintake_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/pageintake"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwi/internal/pagerwi"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/pullintaketest"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/urlhashtest"
)

const (
	offeredPageURL = "https://example.com/"
	landedPageURL  = "http://archive.example/replay/https://example.com/"
	pageTitle      = "Hi"
)

type recordingPageRWIs struct {
	refused   bool
	published []pagerwicontract.PageRWI
}

func (r *recordingPageRWIs) Publish(_ context.Context, pageRWI pagerwicontract.PageRWI) bool {
	if r.refused {
		return false
	}
	r.published = append(r.published, pageRWI)

	return true
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
	disposals         []string
	pagesOffered      int
	pagesNotPublished int
	postingsBuilt     int
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

func (r *recordingPageIntakeObserver) MediaTypeUnsupported(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
	string,
) {
	r.disposals = append(r.disposals, "media_type_unsupported")
}

func (r *recordingPageIntakeObserver) NoIndexDerived(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	r.disposals = append(r.disposals, "no_index_derived")
}

func (r *recordingPageIntakeObserver) PageRWINotPublished(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	r.pagesNotPublished++
}

func (r *recordingPageIntakeObserver) PageRWIBuilt(
	_ context.Context,
	_ string,
	_ canonicalurl.CanonicalURL,
	amountOfPostings int,
) {
	r.disposals = append(r.disposals, "built")
	r.postingsBuilt += amountOfPostings
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
	pageRWIs           pageintake.PageRWIs
	intakeReceipts     pageintake.IntakeReceipts
	pageIntakeObserver pageintake.PageIntakeObserver
}

func run(t *testing.T, msg jetstream.Msg, pageRWIs pageintake.PageRWIs) error {
	return runWith(t, msg, intakeCollaborators{
		pageRWIs:           pageRWIs,
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
			PageIndexBuilder:           indexBuilder(t),
			PageRWIs:                   collaborators.pageRWIs,
			Admission:                  pullintake.EveryItemAdmitted{},
			IntakeReceipts:             collaborators.intakeReceipts,
			PageIntakeObserver:         collaborators.pageIntakeObserver,
			PageOfferIntakeConcurrency: 1,
		}).Run(context.Background())
}

func TestOfferedPageIsBuiltPublishedAndReportedAsKept(t *testing.T) {
	observer := &recordingPageIntakeObserver{}
	receipts := &recordingIntakeReceipts{}
	pageRWIs := &recordingPageRWIs{}
	message := offeredPageMessage(t, "alpha beta")

	if err := runWith(t, message, intakeCollaborators{
		pageRWIs:           pageRWIs,
		intakeReceipts:     receipts,
		pageIntakeObserver: observer,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if got := observer.disposals; len(got) != 1 || got[0] != "built" {
		t.Errorf("disposals = %v, want built", got)
	}
	if len(pageRWIs.published) != 1 {
		t.Fatalf("published %d rwis, want one for the offered page", len(pageRWIs.published))
	}
	published := pageRWIs.published[0]
	if observer.pagesOffered != 1 || observer.postingsBuilt != len(published.Postings) {
		t.Errorf("pages offered = %d and postings built = %d, want one page of %d postings",
			observer.pagesOffered, observer.postingsBuilt, len(published.Postings))
	}
	if published.Metadata.Address != offeredPageURL {
		t.Errorf("published address = %q, want the offered page", published.Metadata.Address)
	}
	if published.Metadata.Title != pageTitle {
		t.Errorf("published title = %q, want the extracted title", published.Metadata.Title)
	}
	if len(receipts.kept) != 1 || receipts.kept[0].String() != offeredPageURL {
		t.Errorf("kept receipts = %v, want one for the offered page", receipts.kept)
	}
	assertWordsPublished(t, pageRWIs, "alpha", "beta")
}

func TestPageWithoutWordsIsPublishedWithoutPostingsAndIsReportedAsKept(t *testing.T) {
	receipts := &recordingIntakeReceipts{}
	pageRWIs := &recordingPageRWIs{}
	message := offeredPage(t, pagescrapecontract.OfferedPage{
		PageURL:     canonicalurltest.CanonicalURLOf(t, offeredPageURL),
		LandedURL:   canonicalurltest.CanonicalURLOf(t, offeredPageURL),
		ContentType: "text/html",
		Body:        []byte(`<html lang="en"><body><p>!!! ...</p></body></html>`),
	})

	if err := runWith(t, message, intakeCollaborators{
		pageRWIs:           pageRWIs,
		intakeReceipts:     receipts,
		pageIntakeObserver: pageintake.PageIntakeObservers{},
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	wantURL := urlhashtest.URLHashOf(t, offeredPageURL)
	if len(pageRWIs.published) != 1 || pageRWIs.published[0].Metadata.Hash != wantURL {
		t.Fatalf("published rwis = %+v, want one for %v", pageRWIs.published, wantURL)
	}
	if postings := pageRWIs.published[0].Postings; len(postings) != 0 {
		t.Errorf("published postings %+v, want none for a page without words", postings)
	}
	if len(receipts.kept) != 1 || receipts.kept[0].String() != offeredPageURL {
		t.Errorf("kept receipts = %v, want one for the offered page", receipts.kept)
	}
}

func TestPageThatLandedElsewhereIsBuiltUnderTheOfferedURL(t *testing.T) {
	pageRWIs := &recordingPageRWIs{}

	message := offeredPage(t, pagescrapecontract.OfferedPage{
		PageURL:     canonicalurltest.CanonicalURLOf(t, offeredPageURL),
		LandedURL:   canonicalurltest.CanonicalURLOf(t, landedPageURL),
		ContentType: "text/html",
		Body:        []byte(`<html lang="en"><body><p>alpha</p></body></html>`),
	})
	if err := run(t, message, pageRWIs); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if len(pageRWIs.published) != 1 ||
		pageRWIs.published[0].Metadata.Address != offeredPageURL {
		t.Fatalf("published %+v, want one rwi under the offered url", pageRWIs.published)
	}
}

func TestConsumerPublishesAPageInOnePageRWI(t *testing.T) {
	pageRWIs := &recordingPageRWIs{}

	if err := run(
		t,
		offeredPageMessage(t, "alpha beta gamma delta epsilon"),
		pageRWIs,
	); err != nil {
		t.Fatalf("run: %v", err)
	}

	assertWordsPublished(t, pageRWIs, "alpha", "beta", "gamma", "delta", "epsilon")
}

func TestPageNoDocumentIsExtractedFromIsReportedAsRejected(t *testing.T) {
	assertRejectedWithDisposal(t, pagescrapecontract.OfferedPage{
		PageURL:     canonicalurltest.CanonicalURLOf(t, offeredPageURL),
		LandedURL:   canonicalurltest.CanonicalURLOf(t, offeredPageURL),
		ContentType: "text/html",
		Body:        []byte{},
	}, "document_extraction_failed")
}

func TestPageOfAnUnsupportedMediaTypeIsReportedAsRejected(t *testing.T) {
	assertRejectedWithDisposal(t, pagescrapecontract.OfferedPage{
		PageURL:     canonicalurltest.CanonicalURLOf(t, offeredPageURL),
		LandedURL:   canonicalurltest.CanonicalURLOf(t, offeredPageURL),
		ContentType: "application/pdf",
		Body:        []byte("%PDF-1.4"),
	}, "media_type_unsupported")
}

func assertRejectedWithDisposal(
	t *testing.T,
	page pagescrapecontract.OfferedPage,
	wantDisposal string,
) {
	t.Helper()

	observer := &recordingPageIntakeObserver{}
	receipts := &recordingIntakeReceipts{}
	pageRWIs := &recordingPageRWIs{}
	message := offeredPage(t, page)
	if err := runWith(t, message, intakeCollaborators{
		pageRWIs:           pageRWIs,
		intakeReceipts:     receipts,
		pageIntakeObserver: observer,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if len(pageRWIs.published) != 0 {
		t.Errorf("published %v, want nothing", pageRWIs.published)
	}
	if len(receipts.rejected) != 1 {
		t.Errorf("rejected receipts = %v, want one for the offered page", receipts.rejected)
	}
	if got := observer.disposals; len(got) != 1 || got[0] != wantDisposal {
		t.Errorf("disposals = %v, want %s", got, wantDisposal)
	}
}

func TestOfferedPageWhosePageRWIIsNotPublishedIsReturnedWithoutAReceipt(t *testing.T) {
	observer := &recordingPageIntakeObserver{}
	receipts := &recordingIntakeReceipts{}
	message := offeredPageMessage(t, "alpha")

	if err := runWith(t, message, intakeCollaborators{
		pageRWIs:           &recordingPageRWIs{refused: true},
		intakeReceipts:     receipts,
		pageIntakeObserver: observer,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.HeldBack {
		t.Errorf("action = %q, want nak", action)
	}
	if observer.pagesNotPublished != 1 || len(observer.disposals) != 0 {
		t.Errorf("pages not published = %d, disposals = %v, want one and none",
			observer.pagesNotPublished, observer.disposals)
	}
	if len(receipts.kept) != 0 || len(receipts.rejected) != 0 {
		t.Errorf("receipts = %+v, want none for a page left for retry", receipts)
	}
}

func TestConsumerHaltsOnAnUndecodableMessage(t *testing.T) {
	msg := &pullintaketest.Message{Body: []byte("not an offered page")}

	err := run(t, msg, &recordingPageRWIs{})

	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Fatalf("err = %v, want a poison message halt", err)
	}
	if settled := msg.Settlements(); len(settled) != 0 {
		t.Errorf("undecodable message settled %v, want it left pending", settled)
	}
}

func indexBuilder(t *testing.T) pagerwi.IndexBuilder {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatal(err)
	}

	return pagerwi.NewIndexBuilder(partitions)
}

func assertWordsPublished(t *testing.T, pageRWIs *recordingPageRWIs, words ...string) {
	t.Helper()

	if len(pageRWIs.published) != 1 {
		t.Fatalf("published %d rwis, want one for the page", len(pageRWIs.published))
	}
	published := map[yacymodel.Hash]bool{}
	for _, posting := range pageRWIs.published[0].Postings {
		published[posting.WordHash] = true
	}
	for _, word := range words {
		if !published[yacymodel.WordHash(word)] {
			t.Errorf("word %q should be published, got %v", word, pageRWIs.published[0].Postings)
		}
	}
}
