package recallgrpc_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerecall"
	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/recallgrpc"
	corpusrecallv1 "github.com/nikitakarpei/yacy-rwi-node/corpusrecallapi/corpusrecall/v1"
)

type fakeRecaller struct {
	kinds  []pagerecall.Kind
	result pagerecall.Result
	err    error
}

func (f *fakeRecaller) Recall(
	_ context.Context,
	_ string,
	kinds []pagerecall.Kind,
) (pagerecall.Result, error) {
	f.kinds = kinds
	return f.result, f.err
}

const (
	kindMarkdown pagerecall.Kind = "markdown"
	kindText     pagerecall.Kind = "text"
)

type markdownPayload struct {
	canonicalURL string
	markdown     string
}

type textPayload struct {
	canonicalURL string
	title        string
	text         string
	language     string
}

func testCodecs() []recallgrpc.RepresentationCodec {
	return []recallgrpc.RepresentationCodec{
		{
			Kind:      kindMarkdown,
			ProtoKind: corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_MARKDOWN,
			Encode: func(representation pagerecall.Representation) *corpusrecallv1.Representation {
				page := representation.(markdownPayload)
				return &corpusrecallv1.Representation{
					Representation: &corpusrecallv1.Representation_Markdown{
						Markdown: &corpusrecallv1.MarkdownRepresentation{
							CanonicalUrl: page.canonicalURL,
							Markdown:     page.markdown,
						},
					},
				}
			},
		},
		{
			Kind:      kindText,
			ProtoKind: corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_TEXT,
			Encode: func(representation pagerecall.Representation) *corpusrecallv1.Representation {
				page := representation.(textPayload)
				return &corpusrecallv1.Representation{
					Representation: &corpusrecallv1.Representation_Text{
						Text: &corpusrecallv1.TextRepresentation{
							CanonicalUrl: page.canonicalURL,
							Title:        page.title,
							Text:         page.text,
							Language:     page.language,
						},
					},
				}
			},
		},
	}
}

func TestRecallTranslatesKindsAndRepresentations(t *testing.T) {
	recaller := &fakeRecaller{result: pagerecall.Result{
		Representations: []pagerecall.RecalledRepresentation{
			{
				Kind:    kindMarkdown,
				Content: markdownPayload{canonicalURL: "https://example.com/", markdown: "# Hi"},
			},
		},
		Unavailable: []pagerecall.Kind{kindText},
	}}
	server := recallgrpc.NewRecallServer(recaller, testCodecs())

	resp, err := server.Recall(context.Background(), &corpusrecallv1.RecallRequest{
		Url: "https://example.com/",
		Kinds: []corpusrecallv1.RepresentationKind{
			corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_MARKDOWN,
			corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_TEXT,
		},
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if want := []pagerecall.Kind{
		kindMarkdown,
		kindText,
	}; len(recaller.kinds) != 2 ||
		recaller.kinds[0] != want[0] ||
		recaller.kinds[1] != want[1] {
		t.Errorf("kinds passed = %v", recaller.kinds)
	}
	if len(resp.GetRepresentations()) != 1 {
		t.Fatalf("representations = %d", len(resp.GetRepresentations()))
	}
	markdown := resp.GetRepresentations()[0].GetMarkdown()
	if markdown == nil || markdown.GetMarkdown() != "# Hi" {
		t.Errorf("markdown representation = %v", markdown)
	}
	if u := resp.GetUnavailable(); len(u) != 1 ||
		u[0] != corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_TEXT {
		t.Errorf("unavailable = %v", u)
	}
}

func TestRecallTranslatesTextRepresentation(t *testing.T) {
	recaller := &fakeRecaller{result: pagerecall.Result{
		Representations: []pagerecall.RecalledRepresentation{
			{Kind: kindText, Content: textPayload{
				canonicalURL: "https://example.com/",
				title:        "Title",
				text:         "body",
				language:     "en",
			}},
		},
	}}
	server := recallgrpc.NewRecallServer(recaller, testCodecs())

	resp, err := server.Recall(context.Background(), &corpusrecallv1.RecallRequest{
		Kinds: []corpusrecallv1.RepresentationKind{
			corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_TEXT,
		},
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	text := resp.GetRepresentations()[0].GetText()
	if text == nil || text.GetTitle() != "Title" || text.GetText() != "body" ||
		text.GetLanguage() != "en" {
		t.Errorf("text representation = %v", text)
	}
}

func TestRecallRejectsUnknownKind(t *testing.T) {
	server := recallgrpc.NewRecallServer(&fakeRecaller{}, testCodecs())

	_, err := server.Recall(context.Background(), &corpusrecallv1.RecallRequest{
		Kinds: []corpusrecallv1.RepresentationKind{
			corpusrecallv1.RepresentationKind_REPRESENTATION_KIND_UNSPECIFIED,
		},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", status.Code(err))
	}
}

func TestRecallMapsInFlightLimitToResourceExhausted(t *testing.T) {
	server := recallgrpc.NewRecallServer(
		&fakeRecaller{err: pagerecall.ErrTooManyInFlight},
		testCodecs(),
	)

	_, err := server.Recall(context.Background(), &corpusrecallv1.RecallRequest{})
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("code = %v, want ResourceExhausted", status.Code(err))
	}
}

func TestRecallMapsOtherFailureToInternal(t *testing.T) {
	server := recallgrpc.NewRecallServer(&fakeRecaller{err: errors.New("boom")}, testCodecs())

	_, err := server.Recall(context.Background(), &corpusrecallv1.RecallRequest{})
	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want Internal", status.Code(err))
	}
}
