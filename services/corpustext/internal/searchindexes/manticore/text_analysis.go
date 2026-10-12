package manticore

import "fmt"

type textAnalysis struct {
	morphology string
	stopwords  string
}

func textAnalysisOf(language string) (textAnalysis, bool) {
	analysis, analyzed := languageTextAnalyses[language]
	return analysis, analyzed
}

func (analysis textAnalysis) tableSettings() string {
	return fmt.Sprintf("morphology='%s' stopwords='%s'", analysis.morphology, analysis.stopwords)
}

var languageTextAnalyses = map[string]textAnalysis{
	"en": {morphology: "stem_en", stopwords: "en"},
	"de": {morphology: "libstemmer_de", stopwords: "de"},
	"fr": {morphology: "libstemmer_fr", stopwords: "fr"},
	"ru": {morphology: "libstemmer_ru", stopwords: "ru"},
}
