package dashboardpage

import (
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"slices"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/prometheus/prometheus/storage"
)

type labelValuesEndpoint struct {
	history storage.Queryable
}

func (e labelValuesEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	selectors, err := selectorsFrom(r)
	if err != nil {
		writeAnswer(w, r, http.StatusBadRequest, failedAnswerOf(errorTypeBadData, err))
		return
	}
	values, err := e.labelValuesOf(r, r.PathValue("name"), selectors)
	if err != nil {
		slog.WarnContext(r.Context(), "dashboard label values failed", slog.Any("error", err))
		writeAnswer(
			w,
			r,
			http.StatusUnprocessableEntity,
			failedAnswerOf(errorTypeExecution, err),
		)
		return
	}
	writeAnswer(w, r, http.StatusOK, answer{Status: statusSuccess, Data: values})
}

func selectorsFrom(r *http.Request) ([][]*labels.Matcher, error) {
	if err := r.ParseForm(); err != nil {
		return nil, fmt.Errorf("form: %w", err)
	}
	asked := r.Form["match[]"]
	if len(asked) == 0 {
		return [][]*labels.Matcher{nil}, nil
	}
	selectors, err := parser.NewParser(parser.Options{}).ParseMetricSelectors(asked)
	if err != nil {
		return nil, fmt.Errorf("match[]: %w", err)
	}
	return selectors, nil
}

func (e labelValuesEndpoint) labelValuesOf(
	r *http.Request,
	name string,
	selectors [][]*labels.Matcher,
) ([]string, error) {
	querier, err := e.history.Querier(math.MinInt64, math.MaxInt64)
	if err != nil {
		return nil, fmt.Errorf("querier: %w", err)
	}
	defer func() { _ = querier.Close() }()
	values := []string{}
	for _, selector := range selectors {
		found, _, err := querier.LabelValues(r.Context(), name, nil, selector...)
		if err != nil {
			return nil, fmt.Errorf("label values: %w", err)
		}
		values = append(values, found...)
	}
	slices.Sort(values)
	return slices.Compact(values), nil
}
