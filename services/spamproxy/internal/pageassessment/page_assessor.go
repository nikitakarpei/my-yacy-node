// Package pageassessment assesses one page for spam with the spam model.
package pageassessment

import (
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spamassessment"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/htmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/spamfeatures"
)

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
) spamassessment.Assessment {
	row := spamfeatures.RowFrom(
		htmlreading.ReadingFrom(
			address,
			body,
			responseHeaders.Get("Content-Type"),
		),
		responseHeaders,
	)
	return spamassessment.Assessment{
		Score:        p.model.ScoreOf(row),
		Threshold:    p.model.Threshold,
		ModelVersion: p.model.Version,
	}
}
