package yacyproto_test

import (
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

func TestSearchRequestCarriesTheRequiredAppearance(t *testing.T) {
	t.Parallel()

	want := yacymodel.Appearance{HasImage: true, AppearsInTitle: true}
	form := yacyproto.SearchRequest{RequiredAppearance: yacymodel.Some(want)}.Form()

	req, err := yacyproto.ParseSearchRequest(t.Context(), form)
	if err != nil {
		t.Fatalf("ParseSearchRequest: %v", err)
	}
	got, ok := req.RequiredAppearance.Get()
	if !ok || got != want {
		t.Fatalf("RequiredAppearance = %+v, %v, want %+v", got, ok, want)
	}
}

func TestSearchRequestReadsEveryEmptyConstraintAsUnconstrained(t *testing.T) {
	t.Parallel()

	allSetBits := yacymodel.Encode([]byte{0xff, 0xff, 0xff, 0xff})
	for _, constraint := range []string{"", "AAAAAA", allSetBits} {
		form := url.Values{yacyproto.FieldConstraint: {constraint}}
		req, err := yacyproto.ParseSearchRequest(t.Context(), form)
		if err != nil {
			t.Fatalf("ParseSearchRequest(%q): %v", constraint, err)
		}
		if _, ok := req.RequiredAppearance.Get(); ok {
			t.Errorf("constraint %q is required, want unconstrained", constraint)
		}
	}
}

func TestSearchRequestOmitsAnAbsentAppearanceConstraint(t *testing.T) {
	t.Parallel()

	form := yacyproto.SearchRequest{
		RequiredAppearance: yacymodel.None[yacymodel.Appearance](),
	}.Form()

	if got := form.Get(yacyproto.FieldConstraint); got != "" {
		t.Fatalf("constraint = %q, want empty", got)
	}
}

func TestSearchRequestRejectsAMalformedAppearanceConstraint(t *testing.T) {
	t.Parallel()

	form := url.Values{yacyproto.FieldConstraint: {"AA=A"}}
	if _, err := yacyproto.ParseSearchRequest(t.Context(), form); err == nil {
		t.Fatal("expected error for malformed constraint")
	}
}
