package dashboardpage_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/tsdb/chunkenc"
	"github.com/prometheus/prometheus/util/annotations"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/dashboardpage"
)

type listedQuerier struct {
	series []storage.Series
}

func (q listedQuerier) Querier(int64, int64) (storage.Querier, error) {
	return q, nil
}

func (q listedQuerier) Select(
	_ context.Context,
	_ bool,
	_ *storage.SelectHints,
	matchers ...*labels.Matcher,
) storage.SeriesSet {
	var matched []storage.Series
	for _, series := range q.series {
		if labels.Selector(matchers).Matches(series.Labels()) {
			matched = append(matched, series)
		}
	}
	return &listedSeries{series: matched, at: -1}
}

func (q listedQuerier) LabelValues(
	_ context.Context, name string, _ *storage.LabelHints, matchers ...*labels.Matcher,
) ([]string, annotations.Annotations, error) {
	var values []string
	for _, series := range q.series {
		if labels.Selector(matchers).Matches(series.Labels()) {
			values = append(values, series.Labels().Get(name))
		}
	}
	return values, nil, nil
}

func (listedQuerier) LabelNames(
	context.Context, *storage.LabelHints, ...*labels.Matcher,
) ([]string, annotations.Annotations, error) {
	return nil, nil, nil
}

func (listedQuerier) Close() error {
	return nil
}

type listedSeries struct {
	series []storage.Series
	at     int
}

func (s *listedSeries) Next() bool {
	s.at++
	return s.at < len(s.series)
}

func (s *listedSeries) At() storage.Series              { return s.series[s.at] }
func (*listedSeries) Err() error                        { return nil }
func (*listedSeries) Warnings() annotations.Annotations { return nil }

func pageServer(t *testing.T) *httptest.Server {
	t.Helper()
	chunk := chunkenc.NewXORChunk()
	appender, err := chunk.Appender()
	if err != nil {
		t.Fatalf("appender: %v", err)
	}
	for sample := range int64(10) {
		appender.Append(0, sample*15_000, float64(sample))
	}
	history := listedQuerier{series: []storage.Series{&storage.SeriesEntry{
		Lset: labels.FromStrings(model.MetricNameLabel, "known_peers", "instance", "yacynode"),
		SampleIteratorFn: func(reused chunkenc.Iterator) chunkenc.Iterator {
			return chunk.Iterator(reused)
		},
	}}}
	mux := http.NewServeMux()
	dashboardpage.Register(mux, history, 15*time.Second, []dashboardpage.Dashboard{
		{Service: "yacynode", Model: []byte(`{"uid":"yacynode","title":"yacynode"}`)},
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server
}

func fetched(t *testing.T, address string, form url.Values) (int, string) {
	t.Helper()
	method := http.MethodGet
	if form != nil {
		method = http.MethodPost
	}
	request, err := http.NewRequestWithContext(
		context.Background(), method, address, strings.NewReader(form.Encode()),
	)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("fetch %s: %v", address, err)
	}
	defer func() { _ = response.Body.Close() }()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read %s: %v", address, err)
	}
	return response.StatusCode, string(body)
}

type rangeAnswer struct {
	Status    string `json:"status"`
	ErrorType string `json:"errorType"`
	Data      struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Values [][2]any          `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

func rangeAnswerOf(t *testing.T, body string) rangeAnswer {
	t.Helper()
	var answer rangeAnswer
	if err := json.Unmarshal([]byte(body), &answer); err != nil {
		t.Fatalf("answer %q: %v", body, err)
	}
	return answer
}

func TestRangeQueryAnswersAsPrometheusDoes(t *testing.T) {
	server := pageServer(t)

	status, body := fetched(t, server.URL+"/dashboards/api/v1/query_range", url.Values{
		"query": {`known_peers{instance=~"(yacynode)"} * 2`},
		"start": {"0"},
		"end":   {"135"},
		"step":  {"15s"},
	})

	answer := rangeAnswerOf(t, body)
	if status != http.StatusOK || answer.Status != "success" || answer.Data.ResultType != "matrix" {
		t.Fatalf("answer = %d %s", status, body)
	}
	if len(answer.Data.Result) != 1 || len(answer.Data.Result[0].Values) != 10 {
		t.Fatalf("result = %s, want one series of 10 points", body)
	}
	if last := answer.Data.Result[0].Values[9]; last[0] != float64(135) || last[1] != "18" {
		t.Errorf("last point = %v, want [135 18]", last)
	}
	if answer.Data.Result[0].Metric["instance"] != "yacynode" {
		t.Errorf("metric = %v, want the instance label", answer.Data.Result[0].Metric)
	}
}

func TestRangeQueryRefusesAQueryItCannotRead(t *testing.T) {
	server := pageServer(t)
	for name, form := range map[string]url.Values{
		"bad expression": {"query": {"rate("}, "start": {"0"}, "end": {"60"}, "step": {"15"}},
		"bad start":      {"query": {"up"}, "start": {"soon"}, "end": {"60"}, "step": {"15"}},
		"bad end":        {"query": {"up"}, "start": {"0"}, "end": {"later"}, "step": {"15"}},
		"bad step":       {"query": {"up"}, "start": {"0"}, "end": {"60"}, "step": {"often"}},
		"no step":        {"query": {"up"}, "start": {"0"}, "end": {"60"}, "step": {"0"}},
		"end first":      {"query": {"up"}, "start": {"60"}, "end": {"0"}, "step": {"15"}},
		"many points":    {"query": {"up"}, "start": {"0"}, "end": {"1000000"}, "step": {"1"}},
	} {
		t.Run(name, func(t *testing.T) {
			status, body := fetched(t, server.URL+"/dashboards/api/v1/query_range", form)
			answer := rangeAnswerOf(t, body)
			if status != http.StatusBadRequest || answer.ErrorType != "bad_data" {
				t.Errorf("answer = %d %s, want 400 bad_data", status, body)
			}
		})
	}
}

func TestRangeQueryReadsInstantsAsRFC3339(t *testing.T) {
	server := pageServer(t)

	status, body := fetched(t, server.URL+"/dashboards/api/v1/query_range", url.Values{
		"query": {"known_peers"},
		"start": {"1970-01-01T00:00:00Z"},
		"end":   {"1970-01-01T00:02:15Z"},
		"step":  {"1m"},
	})

	if answer := rangeAnswerOf(t, body); status != http.StatusOK || len(answer.Data.Result) != 1 {
		t.Errorf("answer = %d %s, want one series", status, body)
	}
}

func TestDashboardListLinksEveryDashboard(t *testing.T) {
	server := pageServer(t)

	status, body := fetched(t, server.URL+"/dashboards", nil)

	if status != http.StatusOK || !strings.Contains(body, `href="/dashboards/yacynode"`) {
		t.Errorf("list = %d %s, want a link to yacynode", status, body)
	}
}

func TestDashboardPageCarriesTheDashboardModel(t *testing.T) {
	server := pageServer(t)

	status, body := fetched(t, server.URL+"/dashboards/yacynode", nil)

	if status != http.StatusOK || !strings.Contains(body, `{"uid":"yacynode","title":"yacynode"}`) {
		t.Errorf("page = %d %s, want the model", status, body)
	}
	if !strings.Contains(body, `data-sample-interval-seconds="15"`) {
		t.Errorf("page = %s, want the sample interval", body)
	}
	for _, asset := range []string{"uPlot.iife.min.js", "uPlot.min.css", "dashboard.js", "dashboard.css"} {
		if assetStatus, _ := fetched(
			t,
			server.URL+"/dashboards/assets/"+asset,
			nil,
		); assetStatus != http.StatusOK {
			t.Errorf("%s = %d, want 200", asset, assetStatus)
		}
	}
}

func TestDashboardPageRefusesAnUnknownService(t *testing.T) {
	server := pageServer(t)

	if status, _ := fetched(
		t,
		server.URL+"/dashboards/spamproxy",
		nil,
	); status != http.StatusNotFound {
		t.Errorf("page = %d, want 404", status)
	}
}

func TestRangeQueryAnswersAnEmptyListWhenNoSeriesMatches(t *testing.T) {
	server := pageServer(t)

	status, body := fetched(t, server.URL+"/dashboards/api/v1/query_range", url.Values{
		"query": {"unknown_metric"},
		"start": {"0"},
		"end":   {"60"},
		"step":  {"15"},
	})

	if status != http.StatusOK || !strings.Contains(body, `"result":[]`) {
		t.Errorf("answer = %d %s, want an empty result list", status, body)
	}
}

func TestLabelValuesListTheValuesOfMatchingSeries(t *testing.T) {
	server := pageServer(t)

	status, body := fetched(
		t,
		server.URL+"/dashboards/api/v1/label/instance/values?match[]="+url.QueryEscape(
			"known_peers",
		),
		nil,
	)

	if status != http.StatusOK || !strings.Contains(body, `"data":["yacynode"]`) {
		t.Errorf("answer = %d %s, want the yacynode instance", status, body)
	}
	if status, _ := fetched(
		t,
		server.URL+"/dashboards/api/v1/label/instance/values?match[]=up{",
		nil,
	); status != http.StatusBadRequest {
		t.Errorf("bad selector answer = %d, want 400", status)
	}
}
