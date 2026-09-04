package pagerefusals_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagehtml"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagerefusals"
)

func refusalsOfHTML(t *testing.T, pageHTML string) pagerefusals.Refusals {
	t.Helper()
	return refusalsOfPage(t, nil, pageHTML)
}

func refusalsOfPage(
	t *testing.T,
	robotsDirectives []string,
	pageHTML string,
) pagerefusals.Refusals {
	t.Helper()
	elementTree, err := pagehtml.NewHTMLParser(silentMediaTypeObserver{}).ElementTreeFrom(
		t.Context(), "text/html", []byte(pageHTML),
	)
	if err != nil {
		t.Fatalf("ElementTreeFrom: %v", err)
	}
	return pagerefusals.RefusalsOfPage(robotsDirectives, elementTree)
}

func TestAPageWithoutMetaRobotsRefusesNothing(t *testing.T) {
	refusals := refusalsOfHTML(t, `<html><body><a href="/next">next</a></body></html>`)

	if refusals.RefusesIndexing || refusals.RefusesLinkDiscovery {
		t.Fatalf("refusals = %+v, want none", refusals)
	}
}

func TestMetaRobotsStatesEachRefusal(t *testing.T) {
	for _, testCase := range []struct {
		content        string
		wantNoIndex    bool
		wantNoDiscover bool
	}{
		{content: "noindex", wantNoIndex: true},
		{content: "nofollow", wantNoDiscover: true},
		{content: "none", wantNoIndex: true, wantNoDiscover: true},
		{content: "NoIndex, NoFollow", wantNoIndex: true, wantNoDiscover: true},
		{content: "index, follow"},
	} {
		refusals := refusalsOfHTML(t,
			`<html><head><meta name="ROBOTS" content="`+testCase.content+`"></head></html>`,
		)

		if refusals.RefusesIndexing != testCase.wantNoIndex ||
			refusals.RefusesLinkDiscovery != testCase.wantNoDiscover {
			t.Errorf("content %q yields %+v", testCase.content, refusals)
		}
	}
}

func TestAMetaTagForSomeoneElseStatesNoRefusal(t *testing.T) {
	refusals := refusalsOfHTML(t,
		`<html><head><meta name="googlebot" content="noindex">`+
			`<meta content="noindex"><meta name="robots"></head></html>`,
	)

	if refusals.RefusesIndexing || refusals.RefusesLinkDiscovery {
		t.Fatalf("refusals = %+v, want none", refusals)
	}
}

func TestDirectivesStatedOutsideTheHTMLRefuseToo(t *testing.T) {
	refusals := refusalsOfPage(t,
		[]string{"noindex", "nofollow"},
		`<html><body><a href="/next">next</a></body></html>`,
	)

	if !refusals.RefusesIndexing || !refusals.RefusesLinkDiscovery {
		t.Fatalf("refusals = %+v, want both", refusals)
	}
}

func TestAPageRefusesWhateverEitherItsDirectivesOrItsHTMLRefuse(t *testing.T) {
	refusals := refusalsOfPage(t,
		[]string{"noindex"},
		`<html><head><meta name="robots" content="nofollow"></head></html>`,
	)

	if !refusals.RefusesIndexing || !refusals.RefusesLinkDiscovery {
		t.Fatalf("refusals = %+v, want both", refusals)
	}
}

type silentMediaTypeObserver struct{}

func (silentMediaTypeObserver) MediaTypeUnparsed(context.Context, string, error) {}
