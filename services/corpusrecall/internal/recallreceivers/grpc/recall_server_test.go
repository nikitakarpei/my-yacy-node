package grpc

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerepresentations/markdown"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recall/pagerecall"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
)

const kindWithoutForm pagerecall.RepresentationKind = "text"

const recalledURL = "https://example.com/"

type fakeRecaller struct {
	kinds  []pagerecall.RepresentationKind
	result pagerecall.RecalledPage
	err    error
}

func (f *fakeRecaller) Recall(
	_ context.Context,
	_ string,
	kinds []pagerecall.RepresentationKind,
) (pagerecall.RecalledPage, error) {
	f.kinds = kinds
	return f.result, f.err
}

type fakeCorpus struct {
	kind pagerecall.RepresentationKind
}

func (c fakeCorpus) RepresentationKind() pagerecall.RepresentationKind { return c.kind }

func (fakeCorpus) RepresentationOf(
	_ context.Context,
	_ string,
) (pagerecall.Representation, bool, error) {
	return nil, false, nil
}

type pageForeignToTheMarkdownForm struct{}

func markdownCorpora() []pagerecall.Corpus {
	return []pagerecall.Corpus{fakeCorpus{kind: markdown.Kind}}
}

func markdownServer(
	t *testing.T,
	recaller Recaller,
) *recallServer {
	t.Helper()
	server, err := newRecallServer(recaller, markdownCorpora())
	if err != nil {
		t.Fatalf("new recall server: %v", err)
	}
	return server
}

func TestRecallAsksForTheKindsTheRequestNames(t *testing.T) {
	recaller := &fakeRecaller{}
	server := markdownServer(t, recaller)

	if _, err := server.Recall(context.Background(), &corpusrecallv1.RecallRequest{
		Url: recalledURL,
		Kinds: []corpusrecallv1.RepresentationKind{
			corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_MARKDOWN,
		},
	}); err != nil {
		t.Fatalf("recall: %v", err)
	}

	if len(recaller.kinds) != 1 || recaller.kinds[0] != markdown.Kind {
		t.Errorf("kinds recalled = %v", recaller.kinds)
	}
}

func TestRecallAnswersWithTheRepresentationsTheRecallYields(t *testing.T) {
	server := markdownServer(t, &fakeRecaller{result: pagerecall.RecalledPage{
		Representations: []pagerecall.RecalledRepresentation{{
			Kind:           markdown.Kind,
			Representation: markdown.Page{CanonicalURL: recalledURL, Markdown: "# Hi"},
		}},
	}})

	response, err := server.Recall(context.Background(), &corpusrecallv1.RecallRequest{
		Url: recalledURL,
	})
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
	server := markdownServer(t, &fakeRecaller{result: pagerecall.RecalledPage{
		UnavailableKinds: []pagerecall.RepresentationKind{markdown.Kind},
	}})

	response, err := server.Recall(context.Background(), &corpusrecallv1.RecallRequest{
		Url: recalledURL,
	})
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
	server := markdownServer(t, &fakeRecaller{})

	_, err := server.Recall(context.Background(), &corpusrecallv1.RecallRequest{
		Kinds: []corpusrecallv1.RepresentationKind{
			corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_TEXT,
		},
	})

	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestRecallFailsWhenARepresentationHasNoFormInTheContract(t *testing.T) {
	server := markdownServer(t, &fakeRecaller{result: pagerecall.RecalledPage{
		Representations: []pagerecall.RecalledRepresentation{{Kind: kindWithoutForm}},
	}})

	_, err := server.Recall(context.Background(), &corpusrecallv1.RecallRequest{})

	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want Internal", status.Code(err))
	}
}

func TestRecallFailsWhenARepresentationIsNotThePageItsFormExpresses(t *testing.T) {
	server := markdownServer(t, &fakeRecaller{result: pagerecall.RecalledPage{
		Representations: []pagerecall.RecalledRepresentation{{
			Kind:           markdown.Kind,
			Representation: pageForeignToTheMarkdownForm{},
		}},
	}})

	_, err := server.Recall(context.Background(), &corpusrecallv1.RecallRequest{})

	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want Internal", status.Code(err))
	}
}

func TestRecallMapsInFlightLimitToResourceExhausted(t *testing.T) {
	server := markdownServer(t, &fakeRecaller{err: pagerecall.ErrTooManyRequestsInFlight})

	_, err := server.Recall(context.Background(), &corpusrecallv1.RecallRequest{})

	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("code = %v, want ResourceExhausted", status.Code(err))
	}
}

func TestRecallMapsOtherFailureToInternal(t *testing.T) {
	server := markdownServer(t, &fakeRecaller{err: errors.New("boom")})

	_, err := server.Recall(context.Background(), &corpusrecallv1.RecallRequest{})

	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want Internal", status.Code(err))
	}
}

func TestARecallServerRefusesACorpusWhoseKindHasNoContractForm(t *testing.T) {
	_, err := newRecallServer(
		&fakeRecaller{},
		[]pagerecall.Corpus{fakeCorpus{kind: kindWithoutForm}},
	)

	if !errors.Is(err, ErrRepresentationKindNotInContract) {
		t.Fatalf("error = %v, want %v", err, ErrRepresentationKindNotInContract)
	}
}
