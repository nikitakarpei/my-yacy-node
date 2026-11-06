package pageintake_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/pagerwicontract"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/pullintake/pullintaketest"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/urlhashtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageintake"
)

const (
	pageAddress           = "https://example.com/"
	nodePartitionExponent = 4
)

type receivedPage struct {
	metadata     yacymodel.URLMetadata
	pagePostings yacymodel.Optional[yacymodel.PagePostings]
}

type recordingPages struct {
	receipt  pageadmission.Receipt
	err      error
	received []receivedPage
}

func (r *recordingPages) Receive(
	_ context.Context,
	metadata yacymodel.URLMetadata,
	pagePostings yacymodel.Optional[yacymodel.PagePostings],
) (pageadmission.Receipt, error) {
	r.received = append(r.received, receivedPage{metadata: metadata, pagePostings: pagePostings})

	return r.receipt, r.err
}

type recordingPageIntakeObserver struct {
	disposals           []string
	pagesReceived       int
	pagesAdmitted       int
	postingsAdmitted    int
	postingsNotAdmitted []int
	admissionFailures   []error
}

func (r *recordingPageIntakeObserver) PageRWIInvalid(context.Context) {
	r.disposals = append(r.disposals, "invalid_message")
}

func (r *recordingPageIntakeObserver) PageRWIReceived(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	r.pagesReceived++
}

func (r *recordingPageIntakeObserver) PostingsRootMismatched(
	context.Context,
	string,
	canonicalurl.CanonicalURL,
) {
	r.disposals = append(r.disposals, "postings_root_mismatch")
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

func partitionsOf(t *testing.T, exponent uint) yacymodel.DHTRingPartitions {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(exponent)
	if err != nil {
		t.Fatal(err)
	}

	return partitions
}

func postingsOf(t *testing.T, words ...string) []yacymodel.RWIPosting {
	t.Helper()

	postings := make([]yacymodel.RWIPosting, 0, len(words))
	for _, word := range words {
		postings = append(postings, yacymodel.RWIPosting{
			WordHash:     yacymodel.WordHash(word),
			URLHash:      urlhashtest.URLHashOf(t, pageAddress),
			DocumentType: yacymodel.DocumentTypeText,
			Language:     yacymodel.LanguageOfUndeclaredDocument,
			Hits:         1,
		})
	}

	return postings
}

func pageRWIBuiltFor(t *testing.T, exponent uint, words ...string) pagerwicontract.PageRWI {
	t.Helper()

	metadata := yacymodel.URLMetadata{
		Hash:    urlhashtest.URLHashOf(t, pageAddress),
		Address: pageAddress,
		Title:   "Hi",
	}
	pageURL := canonicalurltest.CanonicalURLOf(t, pageAddress)
	if len(words) == 0 {
		return pagerwicontract.PageRWI{PageURL: pageURL, Metadata: metadata}
	}
	pagePostings, err := yacymodel.ProvenPagePostingsFrom(
		postingsOf(t, words...), yacymodel.EmptyPostingsProof(partitionsOf(t, exponent)),
	)
	if err != nil {
		t.Fatalf("page postings: %v", err)
	}
	metadata.PostingsRoot = pagePostings.Root()

	return pagerwicontract.PageRWI{
		PageURL:  pageURL,
		Metadata: metadata,
		Postings: pagePostings.Postings(),
	}
}

func pageRWIMessage(t *testing.T, pageRWI pagerwicontract.PageRWI) *pullintaketest.Message {
	t.Helper()

	data, err := pagerwicontract.MarshalPageRWI(pageRWI)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	return &pullintaketest.Message{Body: data}
}

func run(
	t *testing.T,
	message *pullintaketest.Message,
	pages pageadmission.PageReceiver,
	observer pageintake.PageIntakeObserver,
) error {
	t.Helper()

	return pageintake.NewPageRWIConsumer(pageintake.PageRWIConsumerConfig{
		PageRWISource:            pullintaketest.MessageSourceOf(message),
		Partitions:               partitionsOf(t, nodePartitionExponent),
		PageReceiver:             pages,
		PageIntakeObserver:       observer,
		PageRWIIntakeConcurrency: 1,
	}).Run(context.Background())
}

func TestPageRWIIsAdmittedUnderTheRootTheNodeComputes(t *testing.T) {
	observer := &recordingPageIntakeObserver{}
	pages := &recordingPages{}
	pageRWI := pageRWIBuiltFor(t, nodePartitionExponent, "alpha", "beta")
	message := pageRWIMessage(t, pageRWI)

	if err := run(t, message, pages, observer); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if got := observer.disposals; len(got) != 1 || got[0] != "indexed" {
		t.Errorf("disposals = %v, want indexed", got)
	}
	if observer.pagesReceived != 1 || observer.pagesAdmitted != 1 ||
		observer.postingsAdmitted != 2 {
		t.Errorf("observer = %+v, want one page of two postings received and admitted", observer)
	}
	if len(pages.received) != 1 {
		t.Fatalf("admitted %d pages, want one", len(pages.received))
	}
	admitted := pages.received[0]
	if admitted.metadata.Title != "Hi" || admitted.metadata.Address != pageAddress {
		t.Errorf("admitted metadata = %+v, want the metadata of the rwi", admitted.metadata)
	}
	if root := rootOf(admitted.pagePostings); root != pageRWI.Metadata.PostingsRoot {
		t.Errorf("admitted root = %+v, want %+v", root, pageRWI.Metadata.PostingsRoot)
	}
}

func TestPageRWIWithoutPostingsIsAdmittedWithoutPagePostings(t *testing.T) {
	pages := &recordingPages{}
	message := pageRWIMessage(t, pageRWIBuiltFor(t, nodePartitionExponent))

	if err := run(t, message, pages, pageintake.PageIntakeObservers{}); err != nil {
		t.Fatalf("run: %v", err)
	}

	if action := message.Settlement(t); action != pullintaketest.Acknowledged {
		t.Errorf("action = %q, want ack", action)
	}
	if len(pages.received) != 1 || pages.received[0].pagePostings.Present() {
		t.Fatalf("admitted %+v, want one page without page postings", pages.received)
	}
}

func TestPageRWIBuiltForAnotherPartitionExponentHaltsTheIntake(t *testing.T) {
	observer := &recordingPageIntakeObserver{}
	pages := &recordingPages{}
	message := pageRWIMessage(t, pageRWIBuiltFor(t, nodePartitionExponent+1, "alpha"))

	err := run(t, message, pages, observer)

	if !errors.Is(err, pageintake.ErrPostingsRootMismatch) {
		t.Fatalf("err = %v, want a postings root mismatch", err)
	}
	if settled := message.Settlements(); len(settled) != 0 {
		t.Errorf("mismatched rwi settled %v, want it left pending", settled)
	}
	if len(pages.received) != 0 {
		t.Errorf("admitted %+v, want nothing", pages.received)
	}
	if got := observer.disposals; len(got) != 1 || got[0] != "postings_root_mismatch" {
		t.Errorf("disposals = %v, want postings_root_mismatch", got)
	}
}

func TestPageRWIWithARootButNoPostingsHaltsTheIntake(t *testing.T) {
	pageRWI := pageRWIBuiltFor(t, nodePartitionExponent, "alpha")
	pageRWI.Postings = nil

	err := run(t, pageRWIMessage(t, pageRWI), &recordingPages{}, pageintake.PageIntakeObservers{})

	if !errors.Is(err, pageintake.ErrPostingsRootMismatch) {
		t.Fatalf("err = %v, want a postings root mismatch", err)
	}
}

func TestPageRWIThatHoldsAWordTwiceHaltsTheIntake(t *testing.T) {
	observer := &recordingPageIntakeObserver{}
	pageRWI := pageRWIBuiltFor(t, nodePartitionExponent, "alpha")
	pageRWI.Postings = append(pageRWI.Postings, pageRWI.Postings[0])

	err := run(t, pageRWIMessage(t, pageRWI), &recordingPages{}, observer)

	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Fatalf("err = %v, want a poison message halt", err)
	}
	if got := observer.disposals; len(got) != 1 || got[0] != "invalid_message" {
		t.Errorf("disposals = %v, want invalid_message", got)
	}
}

func TestConsumerHaltsOnAnUndecodableMessage(t *testing.T) {
	message := &pullintaketest.Message{Body: []byte("not an rwi")}

	err := run(t, message, &recordingPages{}, pageintake.PageIntakeObservers{})

	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Fatalf("err = %v, want a poison message halt", err)
	}
	if settled := message.Settlements(); len(settled) != 0 {
		t.Errorf("undecodable message settled %v, want it left pending", settled)
	}
}

type admissionDisposalExpectation struct {
	pages            *recordingPages
	wantDisposal     string
	wantFailureCause bool
}

func TestConsumerReportsWhyAdmissionReturnedThePageRWI(t *testing.T) {
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
			message := pageRWIMessage(t, pageRWIBuiltFor(t, nodePartitionExponent, "alpha"))

			if err := run(t, message, expectation.pages, observer); err != nil {
				t.Fatalf("run: %v", err)
			}

			if action := message.Settlement(t); action != pullintaketest.HeldBack {
				t.Errorf("action = %q, want nak", action)
			}
			if got := observer.disposals; len(got) != 1 || got[0] != expectation.wantDisposal {
				t.Errorf("disposals = %v, want %s", got, expectation.wantDisposal)
			}
			if observer.pagesAdmitted != 0 || len(observer.postingsNotAdmitted) != 1 ||
				observer.postingsNotAdmitted[0] != 1 {
				t.Errorf("observer = %+v, want the one posting not admitted", observer)
			}
			if got := len(observer.admissionFailures) != 0; got != expectation.wantFailureCause {
				t.Errorf("failure cause reported = %t, want %t", got, expectation.wantFailureCause)
			}
		})
	}
}

func rootOf(
	pagePostings yacymodel.Optional[yacymodel.PagePostings],
) yacymodel.Optional[yacymodel.PostingsRoot] {
	if pagePostings, present := pagePostings.Get(); present {
		return pagePostings.Root()
	}

	return yacymodel.None[yacymodel.PostingsRoot]()
}
