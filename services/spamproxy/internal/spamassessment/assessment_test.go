package spamassessment_test

import (
	"net/http"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/htmlreading"
	"github.com/nikitakarpei/yacy-rwi-node/spammodel/spamfeatures"
	"github.com/nikitakarpei/yacy-rwi-node/spamproxy/internal/spamassessment"
)

func TestTheHeaderCarriesTheVerdictScoreThresholdAndModel(t *testing.T) {
	assessment := spamassessment.Assessment{Score: 0.93549, Threshold: 0.8, ModelVersion: "2026-09"}

	if want := `spam;score=0.935;threshold=0.8;model="2026-09"`; assessment.HeaderValue() != want {
		t.Fatalf("header = %s, want %s", assessment.HeaderValue(), want)
	}
}

func TestAScoreAtTheThresholdIsClean(t *testing.T) {
	assessment := spamassessment.Assessment{Score: 0.8, Threshold: 0.8, ModelVersion: "2026-09"}

	if assessment.Verdict() != spamassessment.VerdictClean {
		t.Fatalf("verdict = %s, want clean", assessment.Verdict())
	}
}

func TestAPageIsAssessedWithTheScoreOfTheModel(t *testing.T) {
	address, err := canonicalurl.CanonicalURLOf("http://site.example/page")
	if err != nil {
		t.Fatal(err)
	}
	body := []byte("<html><title>Cheap pills</title><p>Buy cheap pills now</p></html>")
	headers := http.Header{"Content-Type": {"text/html"}}
	model := spammodel.Model{
		Version:   "2026-09",
		Threshold: 0.8,
		Intercept: 0.5,
		HashedWeights: map[spamfeatures.Family]spammodel.HashedWeights{
			spamfeatures.Text: spammodel.HashedWeightsFrom(
				indicesOf(t, address, body, headers),
				[]float64{2},
			),
		},
	}

	assessment := spamassessment.NewPageAssessor(model).AssessmentFrom(address, body, headers)

	row := spamfeatures.RowFrom(
		htmlreading.ReadingFrom(address, body, headers.Get("Content-Type")),
		headers,
	)
	want := spamassessment.Assessment{
		Score:        model.ScoreOf(row),
		Threshold:    0.8,
		ModelVersion: "2026-09",
	}
	if assessment != want || assessment.Score == (spammodel.Model{Intercept: 0.5}).ScoreOf(row) {
		t.Fatalf("assessment = %+v, want %+v", assessment, want)
	}
}

func indicesOf(
	t *testing.T,
	address canonicalurl.CanonicalURL,
	body []byte,
	headers http.Header,
) []int32 {
	t.Helper()
	entries := spamfeatures.RowFrom(htmlreading.ReadingFrom(address, body, headers.Get("Content-Type")), headers).HashedEntries[spamfeatures.Text]
	if len(entries) == 0 {
		t.Fatal("the page has no text features")
	}
	return []int32{entries[0].Index}
}
