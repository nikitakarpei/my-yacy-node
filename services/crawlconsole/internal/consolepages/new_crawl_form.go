package consolepages

import (
	"crypto/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

const (
	defaultScope    = yacycrawlcontract.ScopeDomain
	defaultMaxDepth = 2
)

var defaultFields = orderFields{
	Scope:    string(defaultScope),
	MaxDepth: strconv.Itoa(defaultMaxDepth),
}

type newCrawlForm struct {
	crawler  Crawler
	layout   pageLayout
	observer ConsoleObserver
}

type orderFields struct {
	Label           string
	SeedURLs        string
	Scope           string
	MaxDepth        string
	URLMustMatch    string
	URLMustNotMatch string
	AllowQueryURLs  bool
	MaxPagesPerHost string
}

type newCrawlView struct {
	Fields  orderFields
	Scopes  []yacycrawlcontract.CrawlScope
	Problem formProblem
}

type formProblem struct {
	Key   string
	Value string
}

func (form newCrawlForm) show(w http.ResponseWriter, r *http.Request) {
	fields, answered := form.fieldsFor(r)
	form.layout.render(w, r, newCrawlFormScreen, statusOfAnswer(answered), newCrawlView{
		Fields: fields,
		Scopes: yacycrawlcontract.CrawlScopes,
	})
}

func (form newCrawlForm) fieldsFor(r *http.Request) (orderFields, bool) {
	duplicatedCrawlID := r.URL.Query().Get("from")
	if duplicatedCrawlID == "" {
		return defaultFields, true
	}
	crawl, found, err := form.crawler.CrawlOf(r.Context(), duplicatedCrawlID)
	if err != nil {
		form.observer.CrawlerCallFailed(r.Context(), err)
		return defaultFields, false
	}
	if !found || crawl.Order == nil {
		return defaultFields, true
	}
	return orderFieldsFrom(*crawl.Order), true
}

func orderFieldsFrom(order yacycrawlcontract.CrawlOrder) orderFields {
	fields := orderFields{
		Label:           order.Label,
		SeedURLs:        seedLinesOf(order.SeedURLs),
		Scope:           string(order.Profile.Scope),
		MaxDepth:        strconv.Itoa(order.Profile.MaxDepth),
		URLMustMatch:    order.Profile.URLMustMatch,
		URLMustNotMatch: order.Profile.URLMustNotMatch,
		AllowQueryURLs:  order.Profile.AllowQueryURLs,
	}
	if order.Profile.MaxPagesPerHost != yacycrawlcontract.UnlimitedPagesPerHost {
		fields.MaxPagesPerHost = strconv.Itoa(order.Profile.MaxPagesPerHost)
	}
	return fields
}

func seedLinesOf(seedURLs []canonicalurl.CanonicalURL) string {
	lines := make([]string, 0, len(seedURLs))
	for _, seedURL := range seedURLs {
		lines = append(lines, seedURL.String())
	}
	return strings.Join(lines, "\n")
}

func (form newCrawlForm) placeOrder(w http.ResponseWriter, r *http.Request) {
	fields := orderFieldsOf(r)
	order, problem := crawlOrderFrom(fields, rand.Text())
	if problem.Key != "" {
		form.showProblem(w, r, http.StatusBadRequest, fields, problem)
		return
	}
	crawlID, refusal, err := form.crawler.Place(r.Context(), order)
	switch {
	case err != nil:
		form.observer.CrawlerCallFailed(r.Context(), err)
		form.showProblem(w, r, http.StatusBadGateway, fields, formProblem{})
	case refusal != "":
		form.showProblem(w, r, http.StatusBadRequest, fields,
			formProblem{Key: "new-crawl.refused", Value: refusal})
	default:
		returnToCrawl(w, r, crawlID)
	}
}

func orderFieldsOf(r *http.Request) orderFields {
	return orderFields{
		Label:           strings.TrimSpace(r.PostFormValue("Label")),
		SeedURLs:        r.PostFormValue("SeedURLs"),
		Scope:           r.PostFormValue("Scope"),
		MaxDepth:        strings.TrimSpace(r.PostFormValue("MaxDepth")),
		URLMustMatch:    r.PostFormValue("URLMustMatch"),
		URLMustNotMatch: r.PostFormValue("URLMustNotMatch"),
		AllowQueryURLs:  r.PostFormValue("AllowQueryURLs") != "",
		MaxPagesPerHost: strings.TrimSpace(r.PostFormValue("MaxPagesPerHost")),
	}
}

func crawlOrderFrom(
	fields orderFields,
	orderID string,
) (yacycrawlcontract.CrawlOrder, formProblem) {
	seedURLs, problem := seedURLsFrom(fields.SeedURLs)
	if problem.Key != "" {
		return yacycrawlcontract.CrawlOrder{}, problem
	}
	maxDepth, err := strconv.Atoi(fields.MaxDepth)
	if err != nil {
		return yacycrawlcontract.CrawlOrder{},
			formProblem{Key: "new-crawl.max-depth-unreadable", Value: fields.MaxDepth}
	}
	maxPagesPerHost, readable := maxPagesPerHostFrom(fields.MaxPagesPerHost)
	if !readable {
		return yacycrawlcontract.CrawlOrder{},
			formProblem{
				Key:   "new-crawl.max-pages-per-host-unreadable",
				Value: fields.MaxPagesPerHost,
			}
	}
	return yacycrawlcontract.CrawlOrder{
		OrderID:  orderID,
		Label:    fields.Label,
		SeedURLs: seedURLs,
		Profile: yacycrawlcontract.CrawlProfile{
			Scope:           yacycrawlcontract.CrawlScope(fields.Scope),
			URLMustMatch:    fields.URLMustMatch,
			URLMustNotMatch: fields.URLMustNotMatch,
			MaxDepth:        maxDepth,
			AllowQueryURLs:  fields.AllowQueryURLs,
			MaxPagesPerHost: maxPagesPerHost,
		},
	}, formProblem{}
}

func seedURLsFrom(seedLines string) ([]canonicalurl.CanonicalURL, formProblem) {
	var seedURLs []canonicalurl.CanonicalURL
	for line := range strings.Lines(seedLines) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		seedURL, err := canonicalurl.CanonicalURLOf(line)
		if err != nil {
			return nil, formProblem{Key: "new-crawl.seed-url-unreadable", Value: line}
		}
		seedURLs = append(seedURLs, seedURL)
	}
	return seedURLs, formProblem{}
}

func maxPagesPerHostFrom(maxPagesPerHost string) (int, bool) {
	if maxPagesPerHost == "" {
		return yacycrawlcontract.UnlimitedPagesPerHost, true
	}
	amount, err := strconv.Atoi(maxPagesPerHost)
	return amount, err == nil
}

func (form newCrawlForm) showProblem(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	fields orderFields,
	problem formProblem,
) {
	form.layout.render(w, r, newCrawlFormScreen, status, newCrawlView{
		Fields:  fields,
		Scopes:  yacycrawlcontract.CrawlScopes,
		Problem: problem,
	})
}
