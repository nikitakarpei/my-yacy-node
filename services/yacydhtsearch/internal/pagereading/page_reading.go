// Package pagereading reads the pages of the documents one query puts first. One
// run per query starts each page once, at any time until it finishes, and gives
// back, inside one budget, the text, link counts and spam verdict of the pages it
// is asked for. It withdraws the documents whose pages are gone or refuse indexing.
package pagereading

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch"
	"github.com/nikitakarpei/yacy-rwi-node/pagefetch/redirectfollowingfetch"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

type PageToRead struct {
	Document yacymodel.URLHash
	Address  string
}

type PageFetcher interface {
	Fetch(
		ctx context.Context,
		pageURL canonicalurl.CanonicalURL,
		knownVersion pagefetch.PageVersion,
	) (redirectfollowingfetch.LandedFetch, error)
}

type FormatDerivations interface {
	BodyIn(
		ctx context.Context,
		format documentextraction.Format,
		extractedDocument documentextraction.Document,
		pageURL canonicalurl.CanonicalURL,
	) ([]byte, bool)
}

type Clock interface {
	After(timeout time.Duration, expire func()) (stop func())
}

type Reading struct {
	pageReader pageReader
	waiting    pageReadWaiting
	observer   PageReadingObserver
}

//nolint:revive // argument-limit: the reading takes its fetch, formats, budget, cutoff, clock, snippet ceiling and observer
func New(
	pageFetch PageFetcher,
	formatDerivations FormatDerivations,
	pageReadBudget time.Duration,
	cutoff PageReadCutoff,
	clock Clock,
	snippetLengthCeiling int,
	observer PageReadingObserver,
) Reading {
	return Reading{
		pageReader: pageReader{
			pageFetch:            pageFetch,
			formatDerivations:    formatDerivations,
			snippetLengthCeiling: snippetLengthCeiling,
		},
		waiting:  pageReadWaiting{budget: pageReadBudget, cutoff: cutoff, clock: clock},
		observer: observer,
	}
}

func (r Reading) Start(queryWords []yacymodel.Hash) *Run {
	return newRun(r.pageReader, queryWords, r.waiting, r.observer)
}
