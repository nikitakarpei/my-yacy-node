package yacysearchendpoint_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/yacysearchendpoint"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const rankedItems = 50

type recordedRankings struct {
	query   searchquery.Query
	ranking searchresult.Ranking
}

func (r *recordedRankings) RankingFor(
	_ context.Context,
	query searchquery.Query,
) searchresult.Ranking {
	r.query = query

	return r.ranking
}

type searchPage struct {
	Channels []struct {
		Title        string `json:"title"`
		TotalResults string `json:"totalResults"`
		Items        []struct {
			Title       string `json:"title"`
			Link        string `json:"link"`
			Description string `json:"description"`
			PubDate     string `json:"pubDate"`
			Image       string `json:"image"`
		} `json:"items"`
	} `json:"channels"`
}

func answerTo(
	t *testing.T,
	endpoint http.Handler,
	target string,
) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	endpoint.ServeHTTP(
		recorder,
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil),
	)

	return recorder
}

func TestTheEndpointAnswersInYaCysPublicSearchForm(t *testing.T) {
	t.Parallel()

	published := time.Date(2026, time.March, 4, 0, 0, 0, 0, time.UTC)
	rankings := &recordedRankings{ranking: searchresult.Ranking{Items: []searchresult.Item{{
		Address:      "https://example.org/weather",
		Title:        "Weather",
		PublishedAt:  yacymodel.Some(published),
		ImageAddress: "https://example.org/icon.png",
	}}}}

	recorder := answerTo(
		t,
		yacysearchendpoint.New(rankings),
		yacysearchendpoint.Path+"?query=berlin",
	)

	var page searchPage
	if err := json.Unmarshal(recorder.Body.Bytes(), &page); err != nil {
		t.Fatalf("parse answer: %v (body %q)", err, recorder.Body.String())
	}
	if len(page.Channels) != 1 || len(page.Channels[0].Items) != 1 {
		t.Fatalf("answer = %+v, want one channel holding one item", page)
	}
	item := page.Channels[0].Items[0]
	if item.Link != "https://example.org/weather" || item.Title != "Weather" {
		t.Fatalf("item = %+v, want the address and title the network found", item)
	}
	if item.PubDate != published.Format(time.RFC1123Z) {
		t.Fatalf("pubDate = %q, want %q", item.PubDate, published.Format(time.RFC1123Z))
	}
	if page.Channels[0].TotalResults != "1" {
		t.Fatalf("totalResults = %q, want 1", page.Channels[0].TotalResults)
	}
}

func TestTheEndpointReadsTheQueryTheClientAskedFor(t *testing.T) {
	t.Parallel()

	rankings := &recordedRankings{}

	answerTo(
		t,
		yacysearchendpoint.New(rankings),
		yacysearchendpoint.Path+"?query=berlin&startRecord=20&maximumRecords=5&lr=lang_de",
	)

	if len(rankings.query.Terms) != 1 || rankings.query.Terms[0] != "berlin" {
		t.Fatalf("Terms = %v, want berlin", rankings.query.Terms)
	}
	if rankings.query.Language != "lang_de" {
		t.Fatalf("Language = %q, want lang_de", rankings.query.Language)
	}
}

func rankingOver(t *testing.T, addresses ...string) searchresult.Ranking {
	t.Helper()

	items := make([]searchresult.Item, 0, len(addresses))
	for _, address := range addresses {
		hash, err := yacymodel.URLHashOf(address)
		if err != nil {
			t.Fatalf("URLHashOf(%q): %v", address, err)
		}
		items = append(
			items,
			searchresult.ItemFrom(yacymodel.URLMetadata{Hash: hash, Address: address}),
		)
	}

	return searchresult.Ranking{Items: items}
}

func linksIn(t *testing.T, recorder *httptest.ResponseRecorder) []string {
	t.Helper()

	var page searchPage
	if err := json.Unmarshal(recorder.Body.Bytes(), &page); err != nil {
		t.Fatalf("parse answer: %v (body %q)", err, recorder.Body.String())
	}
	if len(page.Channels) != 1 {
		t.Fatalf("answer = %+v, want one channel", page)
	}

	links := make([]string, 0, len(page.Channels[0].Items))
	for _, item := range page.Channels[0].Items {
		links = append(links, item.Link)
	}

	return links
}

func TestTheEndpointCutsThePageTheClientAskedForFromTheRanking(t *testing.T) {
	t.Parallel()

	rankings := &recordedRankings{ranking: rankingOver(
		t, "https://a.example/1", "https://a.example/2", "https://a.example/3",
	)}

	recorder := answerTo(
		t,
		yacysearchendpoint.New(rankings),
		yacysearchendpoint.Path+"?query=berlin&startRecord=1&maximumRecords=2",
	)

	want := []string{"https://a.example/2", "https://a.example/3"}
	if got := linksIn(t, recorder); !slices.Equal(got, want) {
		t.Fatalf("items = %v, want %v", got, want)
	}
}

func TestTwoPagesOfOneRankingCarryDifferentRecords(t *testing.T) {
	t.Parallel()

	rankings := &recordedRankings{ranking: rankingOver(
		t, "https://a.example/1", "https://a.example/2",
	)}
	endpoint := yacysearchendpoint.New(rankings)

	first := linksIn(t, answerTo(
		t, endpoint, yacysearchendpoint.Path+"?query=berlin&startRecord=0&maximumRecords=1",
	))
	second := linksIn(t, answerTo(
		t, endpoint, yacysearchendpoint.Path+"?query=berlin&startRecord=1&maximumRecords=1",
	))

	if !slices.Equal(first, []string{"https://a.example/1"}) ||
		!slices.Equal(second, []string{"https://a.example/2"}) {
		t.Fatalf("pages = %v then %v, want one record each in ranking order", first, second)
	}
}

func TestAPagePastTheLastRecordCarriesNoItem(t *testing.T) {
	t.Parallel()

	rankings := &recordedRankings{ranking: rankingOver(t, "https://a.example/1")}

	recorder := answerTo(
		t,
		yacysearchendpoint.New(rankings),
		yacysearchendpoint.Path+"?query=berlin&startRecord=10",
	)

	if links := linksIn(t, recorder); len(links) != 0 {
		t.Fatalf("items = %v, want none past the last record", links)
	}
}

func manyAddresses(count int) []string {
	addresses := make([]string, 0, count)
	for index := range count {
		addresses = append(addresses, "https://a.example/"+strconv.Itoa(index))
	}

	return addresses
}

func TestAClientThatNamesNoRecordCountGetsTheUsualPage(t *testing.T) {
	t.Parallel()

	rankings := &recordedRankings{ranking: rankingOver(t, manyAddresses(rankedItems)...)}

	recorder := answerTo(
		t,
		yacysearchendpoint.New(rankings),
		yacysearchendpoint.Path+"?query=berlin",
	)

	links := linksIn(t, recorder)
	if len(links) != 10 || links[0] != "https://a.example/0" {
		t.Fatalf("items = %v, want the first ten records", links)
	}
}

func TestAnotherPathIsNotTheSearchEndpoint(t *testing.T) {
	t.Parallel()

	recorder := answerTo(
		t,
		yacysearchendpoint.NewMux(&recordedRankings{}),
		"/elsewhere",
	)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}
