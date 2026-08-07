package elasticsearchindex

type elasticsearchAnalyzer struct {
	analyzerType string
	tokenizer    string
	filters      []string
}

func analyzerOf(language string) elasticsearchAnalyzer {
	analyzer, defined := languageAnalyzers[language]
	if !defined {
		return undeterminedAnalyzer
	}
	return analyzer
}

var undeterminedAnalyzer = elasticsearchAnalyzer{
	analyzerType: "custom",
	tokenizer:    "standard",
	filters:      []string{"lowercase", "asciifolding"},
}

var languageAnalyzers = map[string]elasticsearchAnalyzer{
	"en": {analyzerType: "english"},
	"de": {analyzerType: "german"},
	"fr": {analyzerType: "french"},
	"ru": {analyzerType: "russian"},
}

func (analyzer elasticsearchAnalyzer) definition() map[string]any {
	definition := map[string]any{"type": analyzer.analyzerType}
	if analyzer.tokenizer != "" {
		definition["tokenizer"] = analyzer.tokenizer
		definition["filter"] = analyzer.filters
	}
	return definition
}
