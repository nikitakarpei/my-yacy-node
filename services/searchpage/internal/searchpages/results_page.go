package searchpages

import (
	"bytes"
	"context"
	"embed"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchqueries"
	"github.com/nikitakarpei/yacy-rwi-node/searchpage/internal/searchresults"
)

//go:embed templates
var templateFiles embed.FS

const (
	queryParam                   = "q"
	pageParam                    = "page"
	minimumHighlightedWordLength = 2
)

type resultsPage struct {
	sources        map[searchqueries.Source]ResultSource
	resultLinks    ResultLinks
	observer       PageObserver
	template       *template.Template
	offeredSources []searchqueries.Source
}

type pageView struct {
	Query        string
	PreviousPage string
	NextPage     string
	Operators    []operatorView
	Failures     []string
	Results      []resultView
	PageNumber   int
	Searched     bool
}

type operatorView struct {
	Shortcut string
	Meaning  string
}

type resultView struct {
	Title        string
	Link         string
	ShownAddress string
	Snippet      template.HTML
}

type sourceAnswer struct {
	page     searchresults.SourcePage
	answered bool
}

func newResultsPage(
	sources map[searchqueries.Source]ResultSource,
	resultLinks ResultLinks,
	observer PageObserver,
) resultsPage {
	return resultsPage{
		sources:     sources,
		resultLinks: resultLinks,
		observer:    observer,
		template: template.Must(
			template.New("page.html").ParseFS(templateFiles, "templates/page.html"),
		),
		offeredSources: offeredSourcesIn(sources),
	}
}

func offeredSourcesIn(sources map[searchqueries.Source]ResultSource) []searchqueries.Source {
	var offeredSources []searchqueries.Source
	for _, source := range searchqueries.Sources() {
		if _, offered := sources[source]; offered {
			offeredSources = append(offeredSources, source)
		}
	}
	return offeredSources
}

func (p resultsPage) show(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get(queryParam))
	search := searchqueries.SearchFrom(query, r.URL.Query().Get(pageParam), p.offeredSources)
	p.render(w, r, p.viewFor(r.Context(), query, search))
}

func (p resultsPage) viewFor(
	ctx context.Context,
	query string,
	search searchqueries.Search,
) pageView {
	view := pageView{Query: query, Operators: operatorViewsOf(p.offeredSources)}
	if !search.HasWords() {
		return view
	}
	sourcePages, failedSources := p.sourcePagesFor(ctx, search)
	page := searchresults.PageFrom(sourcePages)
	view.Searched = true
	view.PageNumber = search.PageNumber
	view.Failures = failuresOf(failedSources)
	view.Results = p.resultViewsOf(page.Results, wordPatternOf(search.Words))
	view.PreviousPage, view.NextPage = pageLinksFor(query, search, page.HasNextPage)
	return view
}

func operatorViewsOf(offeredSources []searchqueries.Source) []operatorView {
	views := make([]operatorView, 0, len(offeredSources))
	for _, source := range offeredSources {
		views = append(views, operatorView{
			Shortcut: string(source),
			Meaning:  sourceTexts[source].meaning,
		})
	}
	return views
}

func (p resultsPage) sourcePagesFor(
	ctx context.Context,
	search searchqueries.Search,
) ([]searchresults.SourcePage, []searchqueries.Source) {
	answers := make([]sourceAnswer, len(search.Sources))
	var asks sync.WaitGroup
	for index, source := range search.Sources {
		asks.Go(func() {
			page, answered := p.sources[source].PageFor(ctx, search)
			answers[index] = sourceAnswer{page: page, answered: answered}
		})
	}
	asks.Wait()
	sourcePages := make([]searchresults.SourcePage, 0, len(answers))
	var failedSources []searchqueries.Source
	for index, answer := range answers {
		if !answer.answered {
			failedSources = append(failedSources, search.Sources[index])
			continue
		}
		sourcePages = append(sourcePages, answer.page)
	}
	return sourcePages, failedSources
}

func failuresOf(failedSources []searchqueries.Source) []string {
	failures := make([]string, 0, len(failedSources))
	for _, source := range failedSources {
		failures = append(failures, sourceTexts[source].failure)
	}
	return failures
}

func wordPatternOf(words []string) *regexp.Regexp {
	var quotedWords []string
	for _, word := range words {
		if utf8.RuneCountInString(word) >= minimumHighlightedWordLength {
			quotedWords = append(quotedWords, regexp.QuoteMeta(word))
		}
	}
	if len(quotedWords) == 0 {
		return nil
	}
	slices.SortFunc(quotedWords, func(first, second string) int {
		return len(second) - len(first)
	})
	return regexp.MustCompile("(?i)" + strings.Join(quotedWords, "|"))
}

func (p resultsPage) resultViewsOf(
	results []searchresults.Result,
	wordPattern *regexp.Regexp,
) []resultView {
	views := make([]resultView, 0, len(results))
	for _, result := range results {
		views = append(views, resultView{
			Title:        result.Title,
			Link:         p.resultLinks.LinkFor(result.Address),
			ShownAddress: shownAddressOf(result.Address),
			Snippet:      highlightedIn(result.Snippet, wordPattern),
		})
	}
	return views
}

func shownAddressOf(address string) string {
	shown := address
	for _, prefix := range []string{"https://", "http://", "www."} {
		shown = strings.TrimPrefix(shown, prefix)
	}
	return strings.TrimSuffix(shown, "/")
}

func highlightedIn(text string, wordPattern *regexp.Regexp) template.HTML {
	var html strings.Builder
	end := 0
	if wordPattern != nil {
		for _, match := range wordPattern.FindAllStringIndex(text, -1) {
			html.WriteString(template.HTMLEscapeString(text[end:match[0]]))
			html.WriteString("<b>" + template.HTMLEscapeString(text[match[0]:match[1]]) + "</b>")
			end = match[1]
		}
	}
	html.WriteString(template.HTMLEscapeString(text[end:]))
	return template.HTML(html.String()) //nolint:gosec // G203: every part of the text is escaped.
}

func pageLinksFor(
	query string,
	search searchqueries.Search,
	hasNextPage bool,
) (previousPage string, nextPage string) {
	if !search.IsFirstPage() {
		previousPage = pageLinkFor(query, search.PageNumber-1)
	}
	if hasNextPage {
		nextPage = pageLinkFor(query, search.PageNumber+1)
	}
	return previousPage, nextPage
}

func pageLinkFor(query string, pageNumber int) string {
	form := url.Values{queryParam: {query}}
	if pageNumber > 1 {
		form.Set(pageParam, strconv.Itoa(pageNumber))
	}
	return searchPath + "?" + form.Encode()
}

func (p resultsPage) render(w http.ResponseWriter, r *http.Request, view pageView) {
	var page bytes.Buffer
	if err := p.template.Execute(&page, view); err != nil {
		p.observer.PageRenderFailed(r.Context(), err)
		http.Error(
			w,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = page.WriteTo(w)
}
