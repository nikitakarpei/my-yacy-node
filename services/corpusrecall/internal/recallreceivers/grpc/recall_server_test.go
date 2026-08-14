package grpc_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerepresentations/markdown"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
)

const recalledURL = "https://example.com/"

type fakeRecaller struct {
	mutex  sync.Mutex
	kinds  []recall.RepresentationKind
	result recall.RecalledPage
	err    error
}

func (f *fakeRecaller) Recall(
	_ context.Context,
	_ string,
	kinds []recall.RepresentationKind,
) (recall.RecalledPage, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.kinds = kinds
	return f.result, f.err
}

func (f *fakeRecaller) kindsRecalled() []recall.RepresentationKind {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return f.kinds
}

type pageForeignToTheMarkdownForm struct{}

func TestRecallAsksForTheKindsTheRequestNames(t *testing.T) {
	recaller := &fakeRecaller{}
	receiver := recallReceiverUnderTest(t, recaller)

	if _, err := receiver.Recall(&corpusrecallv1.RecallRequest{
		Url: recalledURL,
		Kinds: []corpusrecallv1.RepresentationKind{
			corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_MARKDOWN,
		},
	}); err != nil {
		t.Fatalf("recall: %v", err)
	}

	kinds := recaller.kindsRecalled()
	if len(kinds) != 1 || kinds[0] != markdown.Kind {
		t.Errorf("kinds recalled = %v", kinds)
	}
}

func TestRecallAnswersWithTheRepresentationsTheRecallYields(t *testing.T) {
	receiver := recallReceiverUnderTest(t, &fakeRecaller{result: recall.RecalledPage{
		Representations: []recall.RecalledRepresentation{{
			Kind:           markdown.Kind,
			Representation: markdown.Page{CanonicalURL: recalledURL, Markdown: "# Hi"},
		}},
	}})

	response, err := receiver.Recall(&corpusrecallv1.RecallRequest{Url: recalledURL})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}

	page := response.GetRepresentations()[0].GetMarkdown()
	if len(response.GetRepresentations()) != 1 ||
		page.GetMarkdown() != "# Hi" ||
		page.GetCanonicalUrl() != recalledURL {
		t.Errorf("representations = %v", response.GetRepresentations())
	}
}

func TestRecallNamesTheKindsTheRecallCouldNotProvide(t *testing.T) {
	receiver := recallReceiverUnderTest(t, &fakeRecaller{result: recall.RecalledPage{
		UnavailableKinds: []recall.RepresentationKind{markdown.Kind},
	}})

	response, err := receiver.Recall(&corpusrecallv1.RecallRequest{Url: recalledURL})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}

	unavailable := response.GetUnavailable()
	if len(unavailable) != 1 ||
		unavailable[0] != corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_MARKDOWN {
		t.Errorf("unavailable = %v", unavailable)
	}
}

func TestRecallRejectsAKindTheServedContractDoesNotName(t *testing.T) {
	receiver := recallReceiverUnderTest(t, &fakeRecaller{})

	_, err := receiver.Recall(&corpusrecallv1.RecallRequest{
		Kinds: []corpusrecallv1.RepresentationKind{
			corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_TEXT,
		},
	})

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestRecallFailsWhenARepresentationHasNoFormInTheContract(t *testing.T) {
	receiver := recallReceiverUnderTest(t, &fakeRecaller{result: recall.RecalledPage{
		Representations: []recall.RecalledRepresentation{{Kind: kindWithoutForm}},
	}})

	_, err := receiver.Recall(&corpusrecallv1.RecallRequest{})

	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want Internal", status.Code(err))
	}
}

func TestRecallFailsWhenARepresentationIsNotThePageItsFormExpresses(t *testing.T) {
	receiver := recallReceiverUnderTest(t, &fakeRecaller{result: recall.RecalledPage{
		Representations: []recall.RecalledRepresentation{{
			Kind:           markdown.Kind,
			Representation: pageForeignToTheMarkdownForm{},
		}},
	}})

	_, err := receiver.Recall(&corpusrecallv1.RecallRequest{})

	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want Internal", status.Code(err))
	}
}

func TestRecallMapsInFlightLimitToResourceExhausted(t *testing.T) {
	receiver := recallReceiverUnderTest(
		t, &fakeRecaller{err: recall.ErrTooManyRequestsInFlight},
	)

	_, err := receiver.Recall(&corpusrecallv1.RecallRequest{})

	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("code = %v, want ResourceExhausted", status.Code(err))
	}
}

func TestRecallMapsOtherFailureToInternal(t *testing.T) {
	receiver := recallReceiverUnderTest(t, &fakeRecaller{err: errors.New("boom")})

	_, err := receiver.Recall(&corpusrecallv1.RecallRequest{})

	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want Internal", status.Code(err))
	}
}
