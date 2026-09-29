// Package spamassessment holds the spam verdict on one page and the header
// that carries it.
package spamassessment

import (
	"net/http"
	"strconv"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/htmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/spamfeatures"
)

const HeaderName = "Spam-Assessment"

const (
	VerdictSpam  = "spam"
	VerdictClean = "clean"
)

type Assessment struct {
	Score        float64
	Threshold    float64
	ModelVersion string
}

func (a Assessment) Verdict() string {
	if a.Score > a.Threshold {
		return VerdictSpam
	}
	return VerdictClean
}

func (a Assessment) HeaderValue() string {
	return a.Verdict() +
		";score=" + strconv.FormatFloat(a.Score, 'f', 3, 64) +
		";threshold=" + strconv.FormatFloat(a.Threshold, 'g', -1, 64) +
		";model=" + strconv.Quote(a.ModelVersion)
}

type PageAssessor struct {
	model spammodel.Model
}

func NewPageAssessor(model spammodel.Model) PageAssessor {
	return PageAssessor{model: model}
}

func (p PageAssessor) AssessmentFrom(
	address canonicalurl.CanonicalURL,
	body []byte,
	responseHeaders http.Header,
) Assessment {
	row := spamfeatures.RowFrom(
		htmlreading.ReadingFrom(
			address,
			body,
			responseHeaders.Get("Content-Type"),
		),
		responseHeaders,
	)
	return Assessment{
		Score:        p.model.ScoreOf(row),
		Threshold:    p.model.Threshold,
		ModelVersion: p.model.Version,
	}
}
