package pendingpagevisit_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/crawlid"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingpagevisit"
)

var crawlID = crawlid.CrawlID{1}

func TestTheSeedPageVisitsOfACrawlStartAtTheTop(t *testing.T) {
	seed := canonicalurltest.CanonicalURLOf(t, "http://host/")

	seeds := pendingpagevisit.SeedPageVisitsOf(crawlID, []canonicalurl.CanonicalURL{seed})

	want := pendingpagevisit.PendingPageVisit{CrawlID: crawlID, URL: seed}
	if len(seeds) != 1 || seeds[0] != want {
		t.Fatalf("seeds = %+v, want %+v", seeds, want)
	}
}

func TestALinkedPageVisitGoesOneLevelDeeperInTheSameCrawl(t *testing.T) {
	visit := pendingpagevisit.PendingPageVisit{
		CrawlID: crawlID,
		URL:     canonicalurltest.CanonicalURLOf(t, "http://host/page"),
		Depth:   3,
	}
	link := canonicalurltest.CanonicalURLOf(t, "http://host/next")

	linked := visit.LinkedPageVisitTo(link)

	want := pendingpagevisit.PendingPageVisit{CrawlID: crawlID, URL: link, Depth: 4}
	if linked != want {
		t.Fatalf("linked %+v, want %+v", linked, want)
	}
}

func TestARedirectedPageVisitKeepsTheDepthInTheSameCrawl(t *testing.T) {
	visit := pendingpagevisit.PendingPageVisit{
		CrawlID: crawlID,
		URL:     canonicalurltest.CanonicalURLOf(t, "http://host/page"),
		Depth:   3,
	}
	redirectTarget := canonicalurltest.CanonicalURLOf(t, "http://host/moved")

	redirected := visit.RedirectedPageVisitTo(redirectTarget)

	want := pendingpagevisit.PendingPageVisit{CrawlID: crawlID, URL: redirectTarget, Depth: 3}
	if redirected != want {
		t.Fatalf("redirected %+v, want %+v", redirected, want)
	}
}
