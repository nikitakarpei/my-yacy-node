// Package pagerecall yields the corpus representations of a page, ordering a crawl of it
// and waiting until the corpus holds them or the recall limit runs out.
package pagerecall

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

var ErrTooManyRequestsInFlight = errors.New("too many recall requests in flight")

type CrawlOrderPlacer interface {
	Place(ctx context.Context, canonicalURL string) error
}

type RedirectResolutions interface {
	ResolvedURLOf(ctx context.Context, canonicalURL string) (string, error)
}

type Corpus interface {
	RepresentationKind() RepresentationKind
	RepresentationOf(ctx context.Context, resolvedURL string) (Representation, bool, error)
}

type DisposedPages interface {
	DisposalOf(ctx context.Context, canonicalURL string) (PageDisposal, error)
}

type PageDisposal interface {
	HasOccurred(ctx context.Context) (bool, error)
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
) *Recaller {
	return &Recaller{
		crawlOrders:      crawlOrders,
		redirects:        redirects,
		disposedPages:    disposedPages,
		corpusByKind:     corpusByKindFrom(corpora),
		progress:         progress,
		recallLimit:      config.RecallLimit,
		pollInterval:     config.PollInterval,
		requestsInFlight: make(chan struct{}, config.MaxRequestsInFlight),
	}
}

func corpusByKindFrom(corpora []Corpus) map[RepresentationKind]Corpus {
	corpusByKind := make(map[RepresentationKind]Corpus, len(corpora))
	for _, corpus := range corpora {
		corpusByKind[corpus.RepresentationKind()] = corpus
	}
	return corpusByKind
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

	canonicalURL, err := canonicalurl.Canonicalize(requestedURL)
	if err != nil {
		return RecalledPage{}, fmt.Errorf("canonicalize %q: %w", requestedURL, err)
	}

	disposal, err := r.disposedPages.DisposalOf(ctx, canonicalURL)
	if err != nil {
		return RecalledPage{}, fmt.Errorf("look up disposal of %q: %w", canonicalURL, err)
	}

	if err := r.crawlOrders.Place(ctx, canonicalURL); err != nil {
		return RecalledPage{}, fmt.Errorf("place crawl order for %q: %w", canonicalURL, err)
	}

	recallCtx, cancel := context.WithTimeout(ctx, r.recallLimit)
	defer cancel()

	var recalled RecalledPage
	for _, kind := range kinds {
		representation, found := r.awaitedRepresentationOf(
			recallCtx,
			canonicalURL,
			kind,
			disposal,
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

func (r *Recaller) awaitedRepresentationOf(
	ctx context.Context,
	canonicalURL string,
	kind RepresentationKind,
	disposal PageDisposal,
) (Representation, bool) {
	corpus, held := r.corpusByKind[kind]
	if !held {
		return nil, false
	}
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()
	for {
		representation, found, err := r.representationOf(ctx, canonicalURL, corpus)
		switch {
		case err != nil:
			slog.WarnContext(ctx, "corpus read failed",
				slog.String("kind", string(kind)),
				slog.String("url", canonicalURL),
				slog.Any("error", err),
			)
			return nil, false
		case found:
			return representation, true
		}
		if disposalHasOccurred(ctx, canonicalURL, disposal) {
			return nil, false
		}
		select {
		case <-ctx.Done():
			return nil, false
		case <-ticker.C:
		}
	}
}

func (r *Recaller) representationOf(
	ctx context.Context,
	canonicalURL string,
	corpus Corpus,
) (Representation, bool, error) {
	resolvedURL, err := r.redirects.ResolvedURLOf(ctx, canonicalURL)
	if err != nil {
		return nil, false, err
	}
	return corpus.RepresentationOf(ctx, resolvedURL)
}

func disposalHasOccurred(
	ctx context.Context,
	canonicalURL string,
	disposal PageDisposal,
) bool {
	disposed, err := disposal.HasOccurred(ctx)
	if err != nil {
		slog.WarnContext(ctx, "disposal lookup failed",
			slog.String("url", canonicalURL),
			slog.Any("error", err),
		)
		return false
	}
	return disposed
}
