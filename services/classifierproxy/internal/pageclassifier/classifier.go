// Package pageclassifier classifies one page with the model of each category.
package pageclassifier

import (
	"net/http"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/classifiermodel"
	"github.com/nikitakarpei/yacy-rwi-node/classifiermodel/htmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/classifiermodel/pagefeatures"
	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
)

type Classifier struct {
	modelPerCategory map[pageclassification.Category]classifiermodel.Model
}

func New(modelPerCategory map[pageclassification.Category]classifiermodel.Model) Classifier {
	return Classifier{modelPerCategory: modelPerCategory}
}

func (c Classifier) ClassificationOf(
	address canonicalurl.CanonicalURL,
	body []byte,
	responseHeaders http.Header,
) pageclassification.Classification {
	row := pagefeatures.RowFrom(
		htmlreading.ReadingFrom(
			address,
			body,
			responseHeaders.Get("Content-Type"),
		),
		responseHeaders,
	)
	classification := make(pageclassification.Classification, len(c.modelPerCategory))
	for category, model := range c.modelPerCategory {
		classification[category] = assessmentBy(model, row)
	}
	return classification
}

func assessmentBy(model classifiermodel.Model, row pagefeatures.Row) pageclassification.Assessment {
	return pageclassification.Assessment{
		Score:        model.ScoreOf(row),
		Threshold:    model.Threshold,
		ModelVersion: model.Version,
	}
}
