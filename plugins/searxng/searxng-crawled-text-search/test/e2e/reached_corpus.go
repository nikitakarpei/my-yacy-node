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
	spanishLanguage   = "es"
	englishAlias      = "english-origin"
	germanAlias       = "german-origin"
	spanishAlias      = "spanish-origin"
	englishURL        = "http://" + englishAlias + "/"
	germanURL         = "http://" + germanAlias + "/"
	spanishURL        = "http://" + spanishAlias + "/"
	englishTitle      = "Riverside Wildflower Guide"
	englishContent    = "A field guide to wildflowers found along riverside trails."
	englishSearchTerm = "wildflower"
	englishStemmed    = "trail"
	germanTitle       = "Wildblumen am Uferweg"
	germanContent     = "Ein Feldfuehrer zu Wildblumen an den Uferwegen."
	germanSearchTerm  = "wildblumen"
	spanishTitle      = "Flores silvestres del sendero"
	spanishContent    = "Una guia de campo de las flores silvestres del sendero del rio."
	spanishSearchTerm = "silvestres"
)

func reachedPageURLs() []string {
	return []string{englishURL, germanURL, spanishURL}
}

func startOrigins(t *testing.T, ctx context.Context, networkName string) {
	t.Helper()
	staticpage.Start(t, ctx, networkName, englishAlias,
		originPage(englishLanguage, englishTitle, englishContent))
	staticpage.Start(t, ctx, networkName, germanAlias,
		originPage(germanLanguage, germanTitle, germanContent))
	staticpage.Start(t, ctx, networkName, spanishAlias,
		originPage(spanishLanguage, spanishTitle, spanishContent))
}

func originPage(language, title, body string) string {
	return `<html lang="` + language + `"><title>` + title + `</title>` +
		`<body><p>` + body + `</p></body></html>`
}
