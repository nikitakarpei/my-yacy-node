// Package recall yields the corpus representations of a page, ordering a crawl of it
// and waiting until the corpus holds them or the recall limit runs out.
package recall

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
)

var (
	ErrTooManyRequestsInFlight       = errors.New("too many recall requests in flight")
	ErrRepresentationKindServedTwice = errors.New(
		"representation kind served by more than one corpus",
	)
)

type CrawlOrderPlacer interface {
	Place(ctx context.Context, canonicalURL yacycrawlcontract.CanonicalURL) error
}

type RedirectResolutions interface {
	ResolvedURLOf(
		ctx context.Context,
		canonicalURL yacycrawlcontract.CanonicalURL,
	) (yacycrawlcontract.CanonicalURL, error)
}

type Corpus interface {
	RepresentationKind() RepresentationKind
	RepresentationOf(
		ctx context.Context,
		resolvedURL yacycrawlcontract.CanonicalURL,
	) (Representation, bool, error)
}

type DisposalMark string

type DisposedPages interface {
	DisposalMarkOf(
		ctx context.Context,
		canonicalURL yacycrawlcontract.CanonicalURL,
	) (DisposalMark, error)
	DisposalOccurredSince(
		ctx context.Context,
		canonicalURL yacycrawlcontract.CanonicalURL,
		mark DisposalMark,
	) (bool, error)
}

type RecallProgress interface {
	RequestAccepted()
	RequestRejected()
	RepresentationRecalled(kind RepresentationKind)
	RepresentationUnavailable(kind RepresentationKind)
}

type Recaller struct {
	crawlOrders      CrawlOrderPlacer
	redirects        RedirectResolutions
	disposedPages    DisposedPages
	corpusByKind     map[RepresentationKind]Corpus
	progress         RecallProgress
	recallLimit      time.Duration
	pollInterval     time.Duration
	requestsInFlight chan struct{}
}

//nolint:revive // argument-limit: six explicit, independently-meaningful collaborators
func NewRecaller(
	crawlOrders CrawlOrderPlacer,
	redirects RedirectResolutions,
	disposedPages DisposedPages,
	corpora []Corpus,
	progress RecallProgress,
	config Config,
) (*Recaller, error) {
	corpusByKind, err := corpusByKindFrom(corpora)
	if err != nil {
		return nil, err
	}
	return &Recaller{
		crawlOrders:      crawlOrders,
		redirects:        redirects,
		disposedPages:    disposedPages,
		corpusByKind:     corpusByKind,
		progress:         progress,
		recallLimit:      config.RecallLimit,
		pollInterval:     config.PollInterval,
		requestsInFlight: make(chan struct{}, config.MaxRequestsInFlight),
	}, nil
}

func corpusByKindFrom(corpora []Corpus) (map[RepresentationKind]Corpus, error) {
	corpusByKind := make(map[RepresentationKind]Corpus, len(corpora))
	for _, corpus := range corpora {
		kind := corpus.RepresentationKind()
		if _, taken := corpusByKind[kind]; taken {
			return nil, fmt.Errorf("%w: %s", ErrRepresentationKindServedTwice, kind)
		}
		corpusByKind[kind] = corpus
	}
	return corpusByKind, nil
}

func (r *Recaller) Recall(
	ctx context.Context,
	requestedURL string,
	kinds []RepresentationKind,
) (RecalledPage, error) {
	select {
	case r.requestsInFlight <- struct{}{}:
		defer func() { <-r.requestsInFlight }()
	default:
		r.progress.RequestRejected()
		return RecalledPage{}, ErrTooManyRequestsInFlight
	}
	r.progress.RequestAccepted()

	canonicalURL, err := yacycrawlcontract.CanonicalURLOf(requestedURL)
	if err != nil {
		return RecalledPage{}, fmt.Errorf("canonicalize %q: %w", requestedURL, err)
	}

	disposalMark, err := r.disposedPages.DisposalMarkOf(ctx, canonicalURL)
	if err != nil {
		return RecalledPage{}, fmt.Errorf("look up disposal mark of %q: %w", canonicalURL, err)
	}

	if err := r.crawlOrders.Place(ctx, canonicalURL); err != nil {
		return RecalledPage{}, fmt.Errorf("place crawl order for %q: %w", canonicalURL, err)
	}

	recallCtx, cancel := context.WithTimeout(ctx, r.recallLimit)
	defer cancel()

	var recalled RecalledPage
	for _, kind := range kinds {
		representation, found := r.representationOf(
			recallCtx,
			canonicalURL,
			kind,
			disposalMark,
		)
		if !found {
			recalled.UnavailableKinds = append(recalled.UnavailableKinds, kind)
			r.progress.RepresentationUnavailable(kind)
			continue
		}
		recalled.Representations = append(recalled.Representations, RecalledRepresentation{
			Kind:           kind,
			Representation: representation,
		})
		r.progress.RepresentationRecalled(kind)
	}
	return recalled, nil
}

func (r *Recaller) representationOf(
	ctx context.Context,
	canonicalURL yacycrawlcontract.CanonicalURL,
	kind RepresentationKind,
	disposalMark DisposalMark,
) (Representation, bool) {
	corpus, held := r.corpusByKind[kind]
	if !held {
		slog.ErrorContext(ctx, "no corpus serves the requested representation kind",
			slog.String("kind", string(kind)),
			slog.String("url", canonicalURL.String()),
		)
		return nil, false
	}
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()
	for {
		representation, found, err := r.representationHeldBy(ctx, corpus, canonicalURL)
		switch {
		case err != nil:
			slog.WarnContext(ctx, "corpus read failed",
				slog.String("kind", string(kind)),
				slog.String("url", canonicalURL.String()),
				slog.Any("error", err),
			)
			return nil, false
		case found:
			return representation, true
		}
		if r.disposalConfirmedSince(ctx, canonicalURL, disposalMark) {
			return nil, false
		}
		select {
		case <-ctx.Done():
			return nil, false
		case <-ticker.C:
		}
	}
}

func (r *Recaller) representationHeldBy(
	ctx context.Context,
	corpus Corpus,
	canonicalURL yacycrawlcontract.CanonicalURL,
) (Representation, bool, error) {
	resolvedURL, err := r.redirects.ResolvedURLOf(ctx, canonicalURL)
	if err != nil {
		return nil, false, err
	}
	return corpus.RepresentationOf(ctx, resolvedURL)
}

func (r *Recaller) disposalConfirmedSince(
	ctx context.Context,
	canonicalURL yacycrawlcontract.CanonicalURL,
	mark DisposalMark,
) bool {
	disposed, err := r.disposedPages.DisposalOccurredSince(ctx, canonicalURL, mark)
	if err != nil {
		slog.WarnContext(ctx, "disposal lookup failed",
			slog.String("url", canonicalURL.String()),
			slog.Any("error", err),
		)
		return false
	}
	return disposed
}
