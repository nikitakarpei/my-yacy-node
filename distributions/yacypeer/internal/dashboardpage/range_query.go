package dashboardpage

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/promql"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/prometheus/prometheus/storage"
)

const (
	queryTimeout       = 2 * time.Minute
	samplesPerQueryMax = 50_000_000
	pointsPerSeriesMax = 11_000
	statusSuccess      = "success"
	statusError        = "error"
	errorTypeBadData   = "bad_data"
	errorTypeExecution = "execution"
	resultTypeMatrix   = "matrix"
)

var (
	errEndBeforeStart  = errors.New("end is before start")
	errStepNotPositive = errors.New("step is not positive")
	errTooManyPoints   = fmt.Errorf(
		"query asks for more than %d points per series",
		pointsPerSeriesMax,
	)
)

type rangeQueryEndpoint struct {
	engine  *promql.Engine
	history storage.Queryable
}

func newRangeQueryEndpoint(history storage.Queryable) rangeQueryEndpoint {
	return rangeQueryEndpoint{
		engine: promql.NewEngine(promql.EngineOpts{
			MaxSamples:           samplesPerQueryMax,
			Timeout:              queryTimeout,
			EnableAtModifier:     true,
			EnableNegativeOffset: true,
		}),
		history: history,
	}
}

type rangeQuery struct {
	start time.Time
	end   time.Time
	expr  string
	step  time.Duration
}

func (e rangeQueryEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	asked, err := rangeQueryFrom(r)
	if err != nil {
		writeAnswer(w, r, http.StatusBadRequest, failedAnswerOf(errorTypeBadData, err))
		return
	}
	query, err := e.engine.NewRangeQuery(
		r.Context(), e.history, nil, asked.expr, asked.start, asked.end, asked.step,
	)
	if err != nil {
		writeAnswer(w, r, http.StatusBadRequest, failedAnswerOf(errorTypeBadData, err))
		return
	}
	defer query.Close()
	result := query.Exec(r.Context())
	if result.Err != nil {
		slog.WarnContext(r.Context(), "dashboard query failed",
			slog.String("query", asked.expr),
			slog.Any("error", result.Err),
		)
		writeAnswer(
			w,
			r,
			http.StatusUnprocessableEntity,
			failedAnswerOf(errorTypeExecution, result.Err),
		)
		return
	}
	warnings, _ := result.Warnings.AsStrings(asked.expr, 0, 0)
	writeAnswer(w, r, http.StatusOK, answer{
		Status:   statusSuccess,
		Data:     &answerData{ResultType: resultTypeMatrix, Result: matrixOf(result.Value)},
		Warnings: warnings,
	})
}

func matrixOf(value parser.Value) promql.Matrix {
	matrix, _ := value.(promql.Matrix)
	if matrix == nil {
		return promql.Matrix{}
	}
	return matrix
}

func rangeQueryFrom(r *http.Request) (rangeQuery, error) {
	start, err := instantFrom(r.FormValue("start"))
	if err != nil {
		return rangeQuery{}, fmt.Errorf("start: %w", err)
	}
	end, err := instantFrom(r.FormValue("end"))
	if err != nil {
		return rangeQuery{}, fmt.Errorf("end: %w", err)
	}
	step, err := durationFrom(r.FormValue("step"))
	if err != nil {
		return rangeQuery{}, fmt.Errorf("step: %w", err)
	}
	switch {
	case end.Before(start):
		return rangeQuery{}, errEndBeforeStart
	case step <= 0:
		return rangeQuery{}, errStepNotPositive
	case end.Sub(start)/step > pointsPerSeriesMax:
		return rangeQuery{}, errTooManyPoints
	}
	return rangeQuery{start: start, end: end, expr: r.FormValue("query"), step: step}, nil
}

func instantFrom(text string) (time.Time, error) {
	if seconds, err := strconv.ParseFloat(text, 64); err == nil {
		whole, fraction := math.Modf(seconds)
		return time.Unix(int64(whole), int64(fraction*float64(time.Second))).UTC(), nil
	}
	instant, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse instant: %w", err)
	}
	return instant, nil
}

func durationFrom(text string) (time.Duration, error) {
	if seconds, err := strconv.ParseFloat(text, 64); err == nil {
		return time.Duration(seconds * float64(time.Second)), nil
	}
	duration, err := model.ParseDuration(text)
	if err != nil {
		return 0, fmt.Errorf("parse duration: %w", err)
	}
	return time.Duration(duration), nil
}

type answer struct {
	Data      any      `json:"data,omitempty"`
	Status    string   `json:"status"`
	ErrorType string   `json:"errorType,omitempty"`
	Error     string   `json:"error,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
}

type answerData struct {
	Result     promql.Matrix `json:"result"`
	ResultType string        `json:"resultType"`
}

func failedAnswerOf(errorType string, err error) answer {
	return answer{Status: statusError, ErrorType: errorType, Error: err.Error()}
}

func writeAnswer(w http.ResponseWriter, r *http.Request, status int, written answer) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(written); err != nil {
		slog.DebugContext(r.Context(), "dashboard reader left", slog.Any("error", err))
	}
}
