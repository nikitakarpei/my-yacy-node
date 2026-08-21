// Package pagescrape fetches one URL and derives the document it holds into one content format.
//
// Scrape returns an error only when a later attempt can succeed: the fetch broke down or the
// origin asked for a delay. Content the library cannot read — an unextractable body, a document
// no derivation reaches — yields no page and no error, because repeating the fetch cannot change
// it.
package pagescrape

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/pagefetch"
)

const (
	msgNothingToScrape     = "fetch holds no content to scrape"
	msgExtractionFailed    = "document extraction failed, page skipped"
	msgDocumentURLRejected = "fetched page url rejected, page skipped"
	msgContentUnderivable  = "document derives no content in the target format, page skipped"
)

type Scraper struct {
	fetcher      pagefetch.Fetcher
	extractor    *contentextraction.DocumentExtractor
	derivations  contentformatgraph.FormatDerivations
	targetFormat contentformatgraph.Format
}

func New(
	fetcher pagefetch.Fetcher,
	targetFormat contentformatgraph.Format,
) (*Scraper, error) {
	extractor, err := contentextraction.New(mediaExtractorsByMediaType())
	if err != nil {
		return nil, err
	}
	derivations := contentformatgraph.New(PageDerivationCatalog())
	if err := derivations.EnsureNoDanglingFormat(
		emittedFormats(),
		[]contentformatgraph.Format{targetFormat},
	); err != nil {
		return nil, err
	}
	return &Scraper{
		fetcher:      fetcher,
		extractor:    extractor,
		derivations:  derivations,
		targetFormat: targetFormat,
	}, nil
}

func (s *Scraper) Scrape(ctx context.Context, pageURL string) (ScrapedPage, bool, error) {
	outcome, err := s.fetcher.Fetch(ctx, pageURL, pagefetch.PageVersion{})
	if err != nil {
		return ScrapedPage{}, false, fmt.Errorf("fetch %s: %w", pageURL, err)
	}
	switch outcome.Status {
	case pagefetch.FetchSucceeded:
		page, scraped := s.pageFrom(ctx, outcome.Page)
		return page, scraped, nil
	case pagefetch.FetchFailed:
		return ScrapedPage{}, false, fmt.Errorf("fetch %s: failed", pageURL)
	case pagefetch.FetchDeferred:
		return ScrapedPage{}, false, fmt.Errorf(
			"fetch %s: deferred for %s",
			pageURL,
			outcome.DeferFor,
		)
	case pagefetch.FetchNotModified, pagefetch.FetchCeased, pagefetch.FetchNotAPage:
		slog.DebugContext(ctx, msgNothingToScrape, slog.String("url", pageURL))
		return ScrapedPage{}, false, nil
	default:
		return ScrapedPage{}, false, fmt.Errorf(
			"fetch %s: unknown status %d",
			pageURL,
			outcome.Status,
		)
	}
}

func (s *Scraper) pageFrom(ctx context.Context, fetched pagefetch.FetchedPage) (ScrapedPage, bool) {
	canonical, err := canonicalurl.Canonicalize(fetched.FinalURL)
	if err != nil {
		slog.WarnContext(ctx, msgDocumentURLRejected,
			slog.String("url", fetched.FinalURL),
			slog.Any("error", err),
		)
		return ScrapedPage{}, false
	}
	document, err := s.extractor.DocumentFrom(
		ctx, canonical, fetched.ContentType, fetched.Body,
	)
	if err != nil {
		slog.WarnContext(ctx, msgExtractionFailed,
			slog.String("url", canonical),
			slog.Any("error", err),
		)
		return ScrapedPage{}, false
	}
	content, derived := s.contentOf(ctx, canonical, document)
	if !derived {
		return ScrapedPage{}, false
	}
	return ScrapedPage{
		CanonicalURL:     canonical,
		Title:            document.Title,
		Language:         document.Language,
		LocalLinks:       document.LocalLinks,
		ExternalLinks:    document.ExternalLinks,
		DocumentByteSize: len(fetched.Body),
		Content:          content,
	}, true
}

func (s *Scraper) contentOf(
	ctx context.Context,
	canonical string,
	document contentextraction.ExtractedDocument,
) ([]byte, bool) {
	content, resolved, err := s.derivations.
		ForPage(canonical, document.Format, document.Body).
		Resolve(s.targetFormat)
	if err != nil || !resolved {
		slog.WarnContext(ctx, msgContentUnderivable,
			slog.String("url", canonical),
			slog.String("format", string(document.Format)),
			slog.String("targetFormat", string(s.targetFormat)),
			slog.Any("error", err),
		)
		return nil, false
	}
	return content, true
}
