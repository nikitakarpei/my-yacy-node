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

type Assessor struct {
	model spammodel.Model
}

func New(model spammodel.Model) Assessor {
	return Assessor{model: model}
}

func (a Assessor) AssessmentFrom(
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
		Score:        a.model.ScoreOf(row),
		Threshold:    a.model.Threshold,
		ModelVersion: a.model.Version,
	}
}
