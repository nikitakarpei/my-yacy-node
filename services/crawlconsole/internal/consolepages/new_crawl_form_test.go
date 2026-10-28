package consolepages_test

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

func orderFields() url.Values {
	return url.Values{
		"Label":           {" docs site "},
		"SeedURLs":        {"https://example.org/docs/\n\n  https://example.org/blog/  \n"},
		"Scope":           {"subpath"},
		"MaxDepth":        {"3"},
		"URLMustMatch":    {"^https://example"},
		"URLMustNotMatch": {"/private/"},
		"AllowQueryURLs":  {"on"},
		"MaxPagesPerHost": {""},
	}
}

func TestNewCrawlFormOffersTheDefaults(t *testing.T) {
	answer := consoleOver(&fakeCrawler{}).get("/crawls/new")

	assertStatus(t, answer, http.StatusOK)
	assertShows(t, answer,
		`value="domain" checked`, `id="MaxDepth" name="MaxDepth" min="0" required value="2"`,
		"Any Page", "Pages in Same Directory",
		`name="RecrawlGrace" min="1" value="" placeholder="any age"`,
		`<option value="days" selected>days</option>`,
	)
}

func TestNewCrawlFormDuplicatesTheOrderOfACrawl(t *testing.T) {
	crawler := &fakeCrawler{
		crawls: map[string]yacycrawlcontract.Crawl{"c1": runningCrawlOf(t, "c1")},
	}

	answer := consoleOver(crawler).get("/crawls/new?from=c1")

	assertShows(t, answer,
		"https://example.org/docs/\nhttps://example.org/blog/</textarea>",
		`value="docs site"`, `value="domain" checked`, `value="3"`, `value="50"`,
	)
}

func TestNewCrawlFormDuplicatesTheRecrawlGraceOfACrawl(t *testing.T) {
	crawl := runningCrawlOf(t, "c1")
	crawl.Order.Profile.RecrawlGrace = 36 * time.Hour
	crawler := &fakeCrawler{crawls: map[string]yacycrawlcontract.Crawl{"c1": crawl}}

	answer := consoleOver(crawler).get("/crawls/new?from=c1")

	assertShows(t, answer,
		`name="RecrawlGrace" min="1" value="36"`, `<option value="hours" selected>hours</option>`,
	)
}

func TestNewCrawlFormOffersTheDefaultsForAnUnknownCrawl(t *testing.T) {
	answer := consoleOver(&fakeCrawler{}).get("/crawls/new?from=c-gone")

	assertStatus(t, answer, http.StatusOK)
	assertShows(t, answer, `value="domain" checked`)
}

func TestNewCrawlFormShowsTheCrawlerDoesNotAnswer(t *testing.T) {
	answer := consoleOver(&fakeCrawler{down: true}).get("/crawls/new?from=c1")

	assertStatus(t, answer, http.StatusBadGateway)
	assertShows(t, answer, "The crawler does not answer.", `value="domain" checked`)
}

func TestPlaceOrderStartsTheCrawlAndShowsIt(t *testing.T) {
	crawler := &fakeCrawler{}

	answer := consoleOver(crawler).post("/crawls", orderFields())

	if len(crawler.placedOrders) != 1 {
		t.Fatalf("placed orders = %v, want one", crawler.placedOrders)
	}
	placed := crawler.placedOrders[0]
	assertRedirect(t, answer, "/crawls/"+crawlIDOf(placed))
	want := yacycrawlcontract.CrawlOrder{
		OrderID: placed.OrderID,
		Label:   "docs site",
		SeedURLs: []canonicalurl.CanonicalURL{
			seedURLOf(t, "https://example.org/docs/"),
			seedURLOf(t, "https://example.org/blog/"),
		},
		Profile: yacycrawlcontract.CrawlProfile{
			Scope:           yacycrawlcontract.ScopeSubpath,
			URLMustMatch:    "^https://example",
			URLMustNotMatch: "/private/",
			MaxDepth:        3,
			AllowQueryURLs:  true,
			MaxPagesPerHost: yacycrawlcontract.UnlimitedPagesPerHost,
		},
	}
	if placed.OrderID == "" || !reflect.DeepEqual(placed, want) {
		t.Fatalf("placed order = %+v, want %+v", placed, want)
	}
}

func TestPlaceOrderGivesEachOrderItsOwnID(t *testing.T) {
	crawler := &fakeCrawler{}
	console := consoleOver(crawler)

	console.post("/crawls", orderFields())
	console.post("/crawls", orderFields())

	if crawler.placedOrders[0].OrderID == crawler.placedOrders[1].OrderID {
		t.Fatalf("both orders got the ID %q", crawler.placedOrders[0].OrderID)
	}
}

func TestPlaceOrderKeepsAPageLimit(t *testing.T) {
	crawler := &fakeCrawler{}
	fields := orderFields()
	fields.Set("MaxPagesPerHost", "40")

	consoleOver(crawler).post("/crawls", fields)

	if limit := crawler.placedOrders[0].Profile.MaxPagesPerHost; limit != 40 {
		t.Fatalf("MaxPagesPerHost = %d, want 40", limit)
	}
}

func TestPlaceOrderKeepsARecrawlGrace(t *testing.T) {
	crawler := &fakeCrawler{}
	fields := orderFields()
	fields.Set("RecrawlGrace", "90")
	fields.Set("RecrawlGraceUnit", "minutes")

	consoleOver(crawler).post("/crawls", fields)

	if grace := crawler.placedOrders[0].Profile.RecrawlGrace; grace != 90*time.Minute {
		t.Fatalf("RecrawlGrace = %s, want 1h30m", grace)
	}
}

func TestPlaceOrderShowsWhyTheCrawlerRefusedIt(t *testing.T) {
	crawler := &fakeCrawler{refusal: "unknown crawl scope"}

	answer := consoleOver(crawler).post("/crawls", orderFields())

	assertStatus(t, answer, http.StatusBadRequest)
	assertShows(t, answer,
		"The crawler refused the crawl: unknown crawl scope",
		"https://example.org/docs/\n\n  https://example.org/blog/  \n</textarea>",
	)
}

func TestPlaceOrderRefusesAFieldItCannotRead(t *testing.T) {
	cases := []struct {
		field   string
		value   string
		problem string
	}{
		{
			"SeedURLs",
			"https://example.org/\nexample.org/docs",
			"The Crawl Start URL example.org/docs is not a web address.",
		},
		{"MaxDepth", "two", "Max Discovery Depth &#34;two&#34; is not a whole number."},
		{"MaxPagesPerHost", "many", "Max Pages per Host &#34;many&#34; is not a whole number."},
		{
			"RecrawlGrace",
			"soon",
			"Re-crawl Pages Older Than &#34;soon&#34; is not a whole number.",
		},
		{
			"RecrawlGraceUnit",
			"weeks",
			"Re-crawl Pages Older Than has no unit &#34;weeks&#34;.",
		},
		{"RecrawlGrace", "31", "Re-crawl Pages Older Than is at most 30 days."},
	}
	for _, tc := range cases {
		t.Run(tc.field+"="+tc.value, func(t *testing.T) {
			crawler := &fakeCrawler{}
			fields := orderFields()
			fields.Set("RecrawlGrace", "2")
			fields.Set("RecrawlGraceUnit", "days")
			fields.Set(tc.field, tc.value)

			answer := consoleOver(crawler).post("/crawls", fields)

			assertStatus(t, answer, http.StatusBadRequest)
			assertShows(t, answer, tc.problem)
			if len(crawler.placedOrders) != 0 {
				t.Fatalf("placed orders = %v, want none", crawler.placedOrders)
			}
		})
	}
}

func TestPlaceOrderShowsTheCrawlerDoesNotAnswer(t *testing.T) {
	console := consoleOver(&fakeCrawler{down: true})

	answer := console.post("/crawls", orderFields())

	assertStatus(t, answer, http.StatusBadGateway)
	assertShows(t, answer, "The crawler does not answer.", `value="docs site"`)
}
