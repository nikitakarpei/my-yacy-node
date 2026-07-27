package pagerecall

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl"
)

var ErrTooManyInFlight = errors.New("too many recall requests in flight")

type Source interface {
	Fetch(ctx context.Context, targetURL string) (Representation, bool, error)
}

type TargetResolver interface {
	Resolve(ctx context.Context, canonicalURL string) (string, error)
}

type CrawlPlacer interface {
	Place(ctx context.Context, canonicalURL string) error
}

type DisposedPages interface {
	Revision(ctx context.Context, canonicalURL string) (uint64, error)
}

type Metrics interface {
	RequestAccepted()
	RequestRejected()
	RepresentationRecalled(kind Kind)
	RepresentationUnavailable(kind Kind)
}

type Recaller struct {
	placer   CrawlPlacer
	resolver TargetResolver
	disposed DisposedPages
	sources  map[Kind]Source
	metrics  Metrics
	deadline time.Duration
	poll     time.Duration
	slots    chan struct{}
}

//nolint:revive // argument-limit: six explicit, independently-meaningful collaborators
func NewRecaller(
	placer CrawlPlacer,
	resolver TargetResolver,
	disposed DisposedPages,
	sources map[Kind]Source,
	metrics Metrics,
	config Config,
) *Recaller {
	return &Recaller{
		placer:   placer,
		resolver: resolver,
		disposed: disposed,
		sources:  sources,
		metrics:  metrics,
		deadline: config.Deadline,
		poll:     config.PollInterval,
		slots:    make(chan struct{}, config.MaxInFlight),
	}
}

func (r *Recaller) Recall(ctx context.Context, rawURL string, kinds []Kind) (Result, error) {
	select {
	case r.slots <- struct{}{}:
		defer func() { <-r.slots }()
	default:
		r.metrics.RequestRejected()
		return Result{}, ErrTooManyInFlight
	}
	r.metrics.RequestAccepted()

	canonicalURL, err := canonicalurl.Canonicalize(rawURL)
	if err != nil {
		return Result{}, fmt.Errorf("canonicalize %q: %w", rawURL, err)
	}

	baselineRevision, err := r.disposed.Revision(ctx, canonicalURL)
	if err != nil {
		return Result{}, fmt.Errorf("read disposal baseline for %q: %w", canonicalURL, err)
	}

	if err := r.placer.Place(ctx, canonicalURL); err != nil {
		return Result{}, fmt.Errorf("place crawl order for %q: %w", canonicalURL, err)
	}

	deadlineCtx, cancel := context.WithTimeout(ctx, r.deadline)
	defer cancel()

	var result Result
	for _, kind := range kinds {
		source, ok := r.sources[kind]
		if !ok {
			result.Unavailable = append(result.Unavailable, kind)
			r.metrics.RepresentationUnavailable(kind)
			continue
		}
		representation, found := r.await(deadlineCtx, kind, source, canonicalURL, baselineRevision)
		if !found {
			result.Unavailable = append(result.Unavailable, kind)
			r.metrics.RepresentationUnavailable(kind)
			continue
		}
		result.Representations = append(result.Representations,
			RecalledRepresentation{Kind: kind, Content: representation})
		r.metrics.RepresentationRecalled(kind)
	}
	return result, nil
}

func (r *Recaller) await(
	ctx context.Context,
	kind Kind,
	source Source,
	canonicalURL string,
	baselineRevision uint64,
) (Representation, bool) {
	ticker := time.NewTicker(r.poll)
	defer ticker.Stop()
	for {
		representation, found, err := r.fetch(ctx, source, canonicalURL)
		switch {
		case err != nil:
			slog.WarnContext(ctx, "recall source fetch failed",
				slog.String("kind", string(kind)),
				slog.String("url", canonicalURL),
				slog.Any("error", err),
			)
			return nil, false
		case found:
			return representation, true
		}
		if r.disposedSinceBaseline(ctx, canonicalURL, baselineRevision) {
			return nil, false
		}
		select {
		case <-ctx.Done():
			return nil, false
		case <-ticker.C:
		}
	}
}

func (r *Recaller) disposedSinceBaseline(
	ctx context.Context,
	canonicalURL string,
	baselineRevision uint64,
) bool {
	revision, err := r.disposed.Revision(ctx, canonicalURL)
	if err != nil {
		slog.WarnContext(ctx, "disposal lookup failed",
			slog.String("url", canonicalURL),
			slog.Any("error", err),
		)
		return false
	}
	return revision > baselineRevision
}

func (r *Recaller) fetch(
	ctx context.Context,
	source Source,
	canonicalURL string,
) (Representation, bool, error) {
	targetURL, err := r.resolver.Resolve(ctx, canonicalURL)
	if err != nil {
		return nil, false, err
	}
	return source.Fetch(ctx, targetURL)
}
