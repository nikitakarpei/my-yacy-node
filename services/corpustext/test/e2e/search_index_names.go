//go:build e2e

package e2e

const (
	indexBaseName     = "yacy_text"
	indexedLanguage   = "en"
	fanOutPrefix      = indexBaseName + "_v1"
	languageIndexName = fanOutPrefix + "_" + indexedLanguage
	fanOutPattern     = fanOutPrefix + "_*"
)
