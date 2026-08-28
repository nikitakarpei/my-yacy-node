package profileadmission_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/profileadmission"
)

func profile(p yacycrawlcontract.CrawlProfile) yacycrawlcontract.CrawlProfile {
	if p.URLMustMatch == "" {
		p.URLMustMatch = yacycrawlcontract.MatchAll
	}
	if p.MaxPagesPerHost == 0 {
		p.MaxPagesPerHost = yacycrawlcontract.UnlimitedPagesPerHost
	}
	return p
}

func TestAdmitsWideScope(t *testing.T) {
	admission, err := profileadmission.New(
		profile(yacycrawlcontract.CrawlProfile{Scope: yacycrawlcontract.ScopeWide, MaxDepth: 2}),
		[]canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, "http://a.com/")},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !admission.Admits(canonicalurltest.CanonicalURLOf(t, "http://other.com/x"), 1) {
		t.Fatal("wide scope should admit any host")
	}
}

func TestAdmitsDomainScope(t *testing.T) {
	admission, _ := profileadmission.New(
		profile(yacycrawlcontract.CrawlProfile{Scope: yacycrawlcontract.ScopeDomain, MaxDepth: 2}),
		[]canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, "http://a.com/")},
	)
	if !admission.Admits(canonicalurltest.CanonicalURLOf(t, "http://a.com/x"), 1) {
		t.Fatal("same host should be admitted")
	}
	if admission.Admits(canonicalurltest.CanonicalURLOf(t, "http://b.com/x"), 1) {
		t.Fatal("other host should be rejected in domain scope")
	}
}

func TestAdmitsSubpathScope(t *testing.T) {
	admission, _ := profileadmission.New(
		profile(yacycrawlcontract.CrawlProfile{Scope: yacycrawlcontract.ScopeSubpath, MaxDepth: 3}),
		[]canonicalurl.CanonicalURL{
			canonicalurltest.CanonicalURLOf(t, "http://a.com/dir/"),
		},
	)
	if !admission.Admits(canonicalurltest.CanonicalURLOf(t, "http://a.com/dir/page"), 1) {
		t.Fatal("subpath child should be admitted")
	}
	if admission.Admits(canonicalurltest.CanonicalURLOf(t, "http://a.com/other/page"), 1) {
		t.Fatal("outside subpath should be rejected")
	}
}

func TestAdmitsDepthLimit(t *testing.T) {
	admission, _ := profileadmission.New(
		profile(yacycrawlcontract.CrawlProfile{Scope: yacycrawlcontract.ScopeWide, MaxDepth: 1}),
		[]canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, "http://a.com/")},
	)
	if admission.Admits(canonicalurltest.CanonicalURLOf(t, "http://a.com/deep"), 2) {
		t.Fatal("beyond max depth should be rejected")
	}
}

func TestAdmitsQueryRejectedWhenDisallowed(t *testing.T) {
	admission, _ := profileadmission.New(
		profile(yacycrawlcontract.CrawlProfile{Scope: yacycrawlcontract.ScopeWide, MaxDepth: 2}),
		[]canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, "http://a.com/")},
	)
	if admission.Admits(canonicalurltest.CanonicalURLOf(t, "http://a.com/x?q=1"), 1) {
		t.Fatal("query URL should be rejected by default")
	}
}

func TestAdmitsMustMatchAndMustNotMatch(t *testing.T) {
	admission, _ := profileadmission.New(
		profile(yacycrawlcontract.CrawlProfile{
			Scope: yacycrawlcontract.ScopeWide, MaxDepth: 2,
			URLMustMatch: `\.html$`, URLMustNotMatch: `/private/`,
		}),
		[]canonicalurl.CanonicalURL{canonicalurltest.CanonicalURLOf(t, "http://a.com/")},
	)
	if !admission.Admits(canonicalurltest.CanonicalURLOf(t, "http://a.com/page.html"), 1) {
		t.Fatal("matching URL should admit")
	}
	if admission.Admits(canonicalurltest.CanonicalURLOf(t, "http://a.com/page.pdf"), 1) {
		t.Fatal("non-matching should reject")
	}
	if admission.Admits(canonicalurltest.CanonicalURLOf(t, "http://a.com/private/x.html"), 1) {
		t.Fatal("must-not-match should reject")
	}
}

func TestNewRejectsBadRegex(t *testing.T) {
	if _, err := profileadmission.New(
		yacycrawlcontract.CrawlProfile{URLMustMatch: "("}, nil,
	); err == nil {
		t.Fatal("bad must-match regex should error")
	}
	if _, err := profileadmission.New(
		yacycrawlcontract.CrawlProfile{URLMustMatch: ".*", URLMustNotMatch: "("}, nil,
	); err == nil {
		t.Fatal("bad must-not-match regex should error")
	}
}
