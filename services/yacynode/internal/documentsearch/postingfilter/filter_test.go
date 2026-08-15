package postingfilter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/postingfilter"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchcriteria"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/documentsearch/searchtest"
)

func postingOfType(kind yacymodel.DocumentType) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{DocumentType: kind}
}

func postingWith(appearance yacymodel.Appearance) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{Appearance: appearance}
}

func TestFilterForReportRejectsOtherSites(t *testing.T) {
	documentHash, err := yacymodel.ParseURLHash("0123456789AB")
	if err != nil {
		t.Fatalf("parse url hash: %v", err)
	}
	posting := yacymodel.RWIPosting{URLHash: documentHash}

	anySite := postingfilter.FilterForReport(searchcriteria.Criteria{})
	if !anySite.Accepts(posting) {
		t.Error("posting should be accepted when no site is requested")
	}
	sameSite := postingfilter.FilterForReport(searchcriteria.Criteria{
		SiteHash: yacymodel.Some(mustHostHash(t, "6789AB")),
	})
	if !sameSite.Accepts(posting) {
		t.Error("posting from the requested site should be accepted")
	}
	otherSite := postingfilter.FilterForReport(searchcriteria.Criteria{
		SiteHash: yacymodel.Some(mustHostHash(t, "000000")),
	})
	if otherSite.Accepts(posting) {
		t.Error("posting from another site should be rejected")
	}
}

func mustHostHash(t *testing.T, s string) yacymodel.HostHash {
	t.Helper()
	hash, err := yacymodel.ParseHostHash(s)
	if err != nil {
		t.Fatalf("ParseHostHash(%q): %v", s, err)
	}

	return hash
}

func TestFilterForReportRequiresContentKindAppearance(t *testing.T) {
	cases := []struct {
		name      string
		kind      searchcriteria.ContentKind
		appearing yacymodel.Appearance
		missing   yacymodel.Appearance
	}{
		{
			"image",
			searchcriteria.ImageContent,
			yacymodel.Appearance{HasImage: true},
			yacymodel.Appearance{HasAudio: true},
		},
		{
			"audio",
			searchcriteria.AudioContent,
			yacymodel.Appearance{HasAudio: true},
			yacymodel.Appearance{HasImage: true},
		},
		{
			"video",
			searchcriteria.VideoContent,
			yacymodel.Appearance{HasVideo: true},
			yacymodel.Appearance{HasImage: true},
		},
		{
			"application",
			searchcriteria.ApplicationContent,
			yacymodel.Appearance{HasApp: true},
			yacymodel.Appearance{HasImage: true},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			filter := postingfilter.FilterForReport(searchcriteria.Criteria{ContentKind: c.kind})
			if !filter.Accepts(postingWith(c.appearing)) {
				t.Error("posting appearing as the requested kind should be accepted")
			}
			if filter.Accepts(postingWith(c.missing)) {
				t.Error("posting without the requested appearance should be rejected")
			}
		})
	}
	anyKind := postingfilter.FilterForReport(
		searchcriteria.Criteria{ContentKind: searchcriteria.AnyContent},
	)
	if !anyKind.Accepts(yacymodel.RWIPosting{}) {
		t.Error("any content kind should accept every posting")
	}
}

func TestFilterForReportRejectsOtherDocumentTypes(t *testing.T) {
	cases := []struct {
		name     string
		kind     searchcriteria.ContentKind
		matching yacymodel.RWIPosting
		other    yacymodel.RWIPosting
	}{
		{
			"image",
			searchcriteria.ImageContent,
			postingOfType(yacymodel.DocumentTypeImage),
			postingOfType(yacymodel.DocumentTypeAudio),
		},
		{
			"audio",
			searchcriteria.AudioContent,
			postingOfType(yacymodel.DocumentTypeAudio),
			postingOfType(yacymodel.DocumentTypeImage),
		},
		{
			"video",
			searchcriteria.VideoContent,
			postingOfType(yacymodel.DocumentTypeMovie),
			postingOfType(yacymodel.DocumentTypeImage),
		},
		{
			"application",
			searchcriteria.ApplicationContent,
			postingWith(yacymodel.Appearance{HasApp: true}),
			postingOfType(yacymodel.DocumentTypeImage),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			filter := postingfilter.FilterForReport(searchcriteria.Criteria{
				ContentKind:       c.kind,
				StrictContentKind: true,
			})
			if !filter.Accepts(c.matching) {
				t.Error("document of the requested type should be accepted")
			}
			if filter.Accepts(c.other) {
				t.Error("document of another type should be rejected")
			}
		})
	}
	anyKind := postingfilter.FilterForReport(searchcriteria.Criteria{StrictContentKind: true})
	if !anyKind.Accepts(postingOfType(yacymodel.DocumentTypeImage)) {
		t.Error("any content kind should accept every document type")
	}
}

func TestFilterForReportRequiresSharedAppearance(t *testing.T) {
	posting := postingWith(yacymodel.Appearance{HasImage: true})

	unconstrained := postingfilter.FilterForReport(searchcriteria.Criteria{})
	if !unconstrained.Accepts(posting) {
		t.Error("posting should be accepted when no appearance is required")
	}
	overlapping := postingfilter.FilterForReport(searchcriteria.Criteria{
		RequiredAppearance: yacymodel.Some(yacymodel.Appearance{HasImage: true}),
	})
	if !overlapping.Accepts(posting) {
		t.Error("posting sharing the required appearance should be accepted")
	}
	disjoint := postingfilter.FilterForReport(searchcriteria.Criteria{
		RequiredAppearance: yacymodel.Some(yacymodel.Appearance{HasVideo: true}),
	})
	if disjoint.Accepts(posting) {
		t.Error("posting without the required appearance should be rejected")
	}
}

func TestFilterForSearchRejectsDocumentsHoldingAnExcludedTerm(t *testing.T) {
	banned := searchtest.HashFor("ban")
	index := searchtest.PostingIndex{Postings: map[yacymodel.Hash][]yacymodel.RWIPosting{
		banned: {{URLHash: searchtest.URLHashFor("url-b")}},
	}}

	filter, err := postingfilter.FilterForSearch(
		context.Background(),
		index,
		searchcriteria.Criteria{ExcludedTerms: []yacymodel.Hash{banned}},
	)
	if err != nil {
		t.Fatalf("FilterForSearch: %v", err)
	}
	if !filter.Accepts(yacymodel.RWIPosting{URLHash: searchtest.URLHashFor("url-a")}) {
		t.Error("document without the excluded term should be accepted")
	}
	if filter.Accepts(yacymodel.RWIPosting{URLHash: searchtest.URLHashFor("url-b")}) {
		t.Error("document holding the excluded term should be rejected")
	}
}

func TestFilterForSearchSurfacesScanFailures(t *testing.T) {
	_, err := postingfilter.FilterForSearch(
		context.Background(),
		searchtest.FailingPostingIndex{Err: errScanBroken},
		searchcriteria.Criteria{ExcludedTerms: []yacymodel.Hash{searchtest.HashFor("ban")}},
	)
	if !errors.Is(err, errScanBroken) {
		t.Fatalf("FilterForSearch error = %v, want %v", err, errScanBroken)
	}
}

var errScanBroken = errors.New("scan broken")

func TestFilterForReportRejectsUnrequiredDocuments(t *testing.T) {
	filter := postingfilter.FilterForReport(searchcriteria.Criteria{
		RequiredDocuments: []yacymodel.URLHash{searchtest.URLHashFor("url-a")},
	})

	if !filter.Accepts(yacymodel.RWIPosting{URLHash: searchtest.URLHashFor("url-a")}) {
		t.Error("required document should be accepted")
	}
	if filter.Accepts(yacymodel.RWIPosting{URLHash: searchtest.URLHashFor("url-b")}) {
		t.Error("document outside the required set should be rejected")
	}
}

func TestFilterForReportRejectsOtherLanguages(t *testing.T) {
	english, err := yacymodel.ParseLanguage("en")
	if err != nil {
		t.Fatalf("ParseLanguage: %v", err)
	}
	german, err := yacymodel.ParseLanguage("de")
	if err != nil {
		t.Fatalf("ParseLanguage: %v", err)
	}
	filter := postingfilter.FilterForReport(
		searchcriteria.Criteria{Language: yacymodel.Some(english)},
	)

	if !filter.Accepts(yacymodel.RWIPosting{Language: yacymodel.Some(english)}) {
		t.Error("posting in the required language should be accepted")
	}
	if filter.Accepts(yacymodel.RWIPosting{Language: yacymodel.Some(german)}) {
		t.Error("posting in another language should be rejected")
	}
	if filter.Accepts(yacymodel.RWIPosting{}) {
		t.Error("posting without a language should be rejected")
	}
}
