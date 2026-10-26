//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/e2eharness/staticpage"
)

const (
	englishLanguage   = "en"
	germanLanguage    = "de"
	englishAlias      = "english-origin"
	germanAlias       = "german-origin"
	englishURL        = "http://" + englishAlias + "/"
	germanURL         = "http://" + germanAlias + "/"
	englishTitle      = "Riverside Wildflower Guide"
	englishContent    = "A field guide to wildflowers found along riverside trails."
	englishSearchTerm = "wildflower"
	germanTitle       = "Wildblumen am Uferweg"
	germanContent     = "Ein Feldfuehrer zu Wildblumen an den Uferwegen."
	germanSearchTerm  = "wildblumen"
)

func scrapedCorpusURLs() []string {
	return []string{englishURL, germanURL}
}

func startOrigins(t *testing.T, ctx context.Context, networkName string) {
	t.Helper()
	staticpage.Start(t, ctx, networkName, englishAlias,
		originPage(englishLanguage, englishTitle, englishContent))
	staticpage.Start(t, ctx, networkName, germanAlias,
		originPage(germanLanguage, germanTitle, germanContent))
}

func originPage(language, title, body string) string {
	return `<html lang="` + language + `"><title>` + title + `</title>` +
		`<body><p>` + body + `</p></body></html>`
}
