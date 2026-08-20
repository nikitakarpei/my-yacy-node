//go:build e2e

package e2e

import (
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract/canonicalurltest"
)

func crawledPages(t *testing.T) []yacycrawlcontract.PageTextRepresentation {
	return []yacycrawlcontract.PageTextRepresentation{
		{
			PageReference: yacycrawlcontract.PageReference{
				CanonicalURL: canonicalurltest.CanonicalURLOf(t, englishURL),
				Title:        englishTitle,
				CrawledAt:    time.Now().UTC(),
				Language:     englishLanguage,
			},
			Text: []byte(englishContent),
		},
		{
			PageReference: yacycrawlcontract.PageReference{
				CanonicalURL: canonicalurltest.CanonicalURLOf(t, germanURL),
				Title:        germanTitle,
				CrawledAt:    time.Now().UTC(),
				Language:     germanLanguage,
			},
			Text: []byte(germanContent),
		},
		{
			PageReference: yacycrawlcontract.PageReference{
				CanonicalURL: canonicalurltest.CanonicalURLOf(t, spanishURL),
				Title:        spanishTitle,
				CrawledAt:    time.Now().UTC(),
				Language:     spanishLanguage,
			},
			Text: []byte(spanishContent),
		},
	}
}

const (
	englishLanguage   = "en"
	germanLanguage    = "de"
	englishTitle      = "Riverside Wildflower Guide"
	englishURL        = "https://example.invalid/wildflower-guide"
	englishContent    = "A field guide to wildflowers found along riverside trails."
	englishSearchTerm = "wildflower"
	englishStemmed    = "trail"
	germanTitle       = "Wildblumen am Uferweg"
	germanURL         = "https://example.invalid/wildblumen-uferweg"
	germanContent     = "Ein Feldfuehrer zu Wildblumen an den Uferwegen."
	germanSearchTerm  = "wildblumen"
	spanishLanguage   = "es"
	spanishTitle      = "Flores silvestres del sendero"
	spanishURL        = "https://example.invalid/flores-silvestres"
	spanishContent    = "Una guia de campo de las flores silvestres del sendero del rio."
	spanishSearchTerm = "silvestres"
)
