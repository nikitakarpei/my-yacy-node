package pageclassifier_test

import (
	"maps"
	"net/http"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/classifiermodel"
	"github.com/nikitakarpei/yacy-rwi-node/classifiermodel/htmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/classifiermodel/pagefeatures"
	"github.com/nikitakarpei/yacy-rwi-node/classifierproxy/internal/pageclassifier"
	"github.com/nikitakarpei/yacy-rwi-node/pageclassification"
)

func TestEachCategoryIsAssessedWithTheScoreOfItsModel(t *testing.T) {
	address, err := canonicalurl.CanonicalURLOf("http://site.example/page")
	if err != nil {
		t.Fatal(err)
	}
	body := []byte("<html><title>Cheap pills</title><p>Buy cheap pills now</p></html>")
	headers := http.Header{"Content-Type": {"text/html"}}
	row := pagefeatures.RowFrom(
		htmlreading.ReadingFrom(address, body, headers.Get("Content-Type")),
		headers,
	)
	spamModel := modelWeighingTheFirstTextFeatureOf(t, row, 2, "2026-09")
	adultModel := modelWeighingTheFirstTextFeatureOf(t, row, -2, "2026-11")

	classification := pageclassifier.New(map[pageclassification.Category]classifiermodel.Model{
		pageclassification.Spam: spamModel,
		"adult":                 adultModel,
	}).ClassificationOf(address, body, headers)

	if want := (pageclassification.Classification{
		pageclassification.Spam: {
			Score: spamModel.ScoreOf(row), Threshold: 0.5, ModelVersion: "2026-09",
		},
		"adult": {Score: adultModel.ScoreOf(row), Threshold: 0.5, ModelVersion: "2026-11"},
	}); !maps.Equal(classification, want) ||
		classification[pageclassification.Spam].Verdict() != pageclassification.Positive ||
		classification["adult"].Verdict() != pageclassification.Negative {
		t.Fatalf("classification = %+v, want %+v", classification, want)
	}
}

func modelWeighingTheFirstTextFeatureOf(
	t *testing.T,
	row pagefeatures.Row,
	weight float64,
	version string,
) classifiermodel.Model {
	t.Helper()
	entries := row.HashedEntries[pagefeatures.Text]
	if len(entries) == 0 {
		t.Fatal("the page has no text features")
	}
	return classifiermodel.Model{
		Version:   version,
		Threshold: 0.5,
		HashedWeights: map[pagefeatures.Family]classifiermodel.HashedWeights{
			pagefeatures.Text: classifiermodel.HashedWeightsFrom(
				[]int32{entries[0].Index},
				[]float64{weight},
			),
		},
	}
}
