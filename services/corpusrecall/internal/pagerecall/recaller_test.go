package pagerecall_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/corpusrecall/internal/pagerecall"
)

const (
	kindMarkdown pagerecall.Kind = "markdown"
	kindText     pagerecall.Kind = "text"
)

type fakePage struct {
	kind pagerecall.Kind
}

func (p fakePage) Kind() pagerecall.Kind { return p.kind }

type recordingPlacer struct {
	placed []string
	err    error
}

func (p *recordingPlacer) Place(_ context.Context, canonicalURL string) error {
	p.placed = append(p.placed, canonicalURL)
	return p.err
}

type identityResolver struct {
	err error
}

func (r identityResolver) Resolve(_ context.Context, canonicalURL string) (string, error) {
	return canonicalURL, r.err
}

type scriptedSource struct {
	kind    pagerecall.Kind
	mu      sync.Mutex
	calls   int
	readyAt int
	page    pagerecall.Representation
	err     error
}

func sources(list ...*scriptedSource) map[pagerecall.Kind]pagerecall.Source {
	byKind := make(map[pagerecall.Kind]pagerecall.Source, len(list))
	for _, source := range list {
		byKind[source.kind] = source
	}
	return byKind
}

func (s *scriptedSource) Fetch(
	_ context.Context,
	_ string,
) (pagerecall.Representation, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.err != nil {
		return nil, false, s.err
	}
	if s.calls >= s.readyAt {
		return s.page, true, nil
	}
	return nil, false, nil
}

type fakeDisposedPages struct {
	mu       sync.Mutex
	revision uint64
	err      error
}

func (d *fakeDisposedPages) Revision(_ context.Context, _ string) (uint64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.revision, d.err
}

type steppingDisposedPages struct {
	mu         sync.Mutex
	calls      int
	revisionAt int
}

func (d *steppingDisposedPages) Revision(_ context.Context, _ string) (uint64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.calls++
	if d.calls >= d.revisionAt {
		return 1, nil
	}
	return 0, nil
}

type countingMetrics struct {
	accepted, rejected int
	recalled           []pagerecall.Kind
	unavailable        []pagerecall.Kind
}

func (m *countingMetrics) RequestAccepted() { m.accepted++ }
func (m *countingMetrics) RequestRejected() { m.rejected++ }
func (m *countingMetrics) RepresentationRecalled(kind pagerecall.Kind) {
	m.recalled = append(m.recalled, kind)
}

func (m *countingMetrics) RepresentationUnavailable(kind pagerecall.Kind) {
	m.unavailable = append(m.unavailable, kind)
}

func markdownSource(readyAt int) *scriptedSource {
	return &scriptedSource{
		kind:    kindMarkdown,
		readyAt: readyAt,
		page:    fakePage{kind: kindMarkdown},
	}
}

func TestRecallReturnsPresentRepresentation(t *testing.T) {
	placer := &recordingPlacer{}
	metrics := &countingMetrics{}
	recaller := pagerecall.NewRecaller(
		placer,
		identityResolver{},
		&fakeDisposedPages{},
		sources(markdownSource(1)),
		metrics,
		pagerecall.Config{Deadline: time.Second, PollInterval: time.Millisecond, MaxInFlight: 4},
	)

	result, err := recaller.Recall(
		context.Background(), "https://example.com", []pagerecall.Kind{kindMarkdown},
	)
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(result.Representations) != 1 || len(result.Unavailable) != 0 {
		t.Fatalf("result = %+v", result)
	}
	if len(placer.placed) != 1 {
		t.Errorf("placed = %v", placer.placed)
	}
	if metrics.accepted != 1 || len(metrics.recalled) != 1 {
		t.Errorf("metrics = %+v", metrics)
	}
}

func TestRecallPollsUntilRepresentationAppears(t *testing.T) {
	source := markdownSource(3)
	recaller := pagerecall.NewRecaller(
		&recordingPlacer{},
		identityResolver{},
		&fakeDisposedPages{},
		sources(source),
		&countingMetrics{},
		pagerecall.Config{Deadline: time.Second, PollInterval: time.Millisecond, MaxInFlight: 4},
	)

	result, err := recaller.Recall(
		context.Background(), "https://example.com", []pagerecall.Kind{kindMarkdown},
	)
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(result.Representations) != 1 {
		t.Fatalf("result = %+v", result)
	}
	if source.calls < 3 {
		t.Errorf("calls = %d, want at least 3", source.calls)
	}
}

func TestRecallReportsUnknownKindUnavailable(t *testing.T) {
	metrics := &countingMetrics{}
	recaller := pagerecall.NewRecaller(
		&recordingPlacer{},
		identityResolver{},
		&fakeDisposedPages{},
		sources(markdownSource(1)),
		metrics,
		pagerecall.Config{Deadline: time.Second, PollInterval: time.Millisecond, MaxInFlight: 4},
	)

	result, err := recaller.Recall(
		context.Background(), "https://example.com", []pagerecall.Kind{kindText},
	)
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(result.Representations) != 0 ||
		len(result.Unavailable) != 1 || result.Unavailable[0] != kindText {
		t.Fatalf("result = %+v", result)
	}
	if len(metrics.unavailable) != 1 {
		t.Errorf("unavailable metric = %v", metrics.unavailable)
	}
}

func TestRecallReportsUnavailableAtDeadline(t *testing.T) {
	metrics := &countingMetrics{}
	recaller := pagerecall.NewRecaller(
		&recordingPlacer{},
		identityResolver{},
		&fakeDisposedPages{},
		sources(markdownSource(1_000_000)),
		metrics,
		pagerecall.Config{
			Deadline:     20 * time.Millisecond,
			PollInterval: time.Millisecond,
			MaxInFlight:  4,
		},
	)

	result, err := recaller.Recall(
		context.Background(), "https://example.com", []pagerecall.Kind{kindMarkdown},
	)
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(result.Unavailable) != 1 {
		t.Fatalf("result = %+v", result)
	}
}

func TestRecallGivesUpKindOnSourceError(t *testing.T) {
	source := &scriptedSource{kind: kindMarkdown, err: errors.New("store down")}
	recaller := pagerecall.NewRecaller(
		&recordingPlacer{},
		identityResolver{},
		&fakeDisposedPages{},
		sources(source),
		&countingMetrics{},
		pagerecall.Config{Deadline: time.Second, PollInterval: time.Millisecond, MaxInFlight: 4},
	)

	result, err := recaller.Recall(
		context.Background(), "https://example.com", []pagerecall.Kind{kindMarkdown},
	)
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(result.Unavailable) != 1 {
		t.Fatalf("result = %+v", result)
	}
}

func TestRecallRejectsWhenInFlightLimitReached(t *testing.T) {
	release := make(chan struct{})
	blocking := &blockingPlacer{entered: make(chan struct{}), release: release}
	metrics := &countingMetrics{}
	recaller := pagerecall.NewRecaller(
		blocking,
		identityResolver{},
		&fakeDisposedPages{},
		sources(markdownSource(1)),
		metrics,
		pagerecall.Config{Deadline: time.Second, PollInterval: time.Millisecond, MaxInFlight: 1},
	)

	go func() {
		_, _ = recaller.Recall(
			context.Background(), "https://example.com", []pagerecall.Kind{kindMarkdown},
		)
	}()
	<-blocking.entered

	_, err := recaller.Recall(
		context.Background(), "https://example.com", []pagerecall.Kind{kindMarkdown},
	)
	if !errors.Is(err, pagerecall.ErrTooManyInFlight) {
		t.Fatalf("err = %v, want ErrTooManyInFlight", err)
	}
	if metrics.rejected != 1 {
		t.Errorf("rejected = %d", metrics.rejected)
	}
	close(release)
}

func TestRecallFailsOnUncanonicalizableURL(t *testing.T) {
	recaller := pagerecall.NewRecaller(
		&recordingPlacer{},
		identityResolver{},
		&fakeDisposedPages{},
		nil,
		&countingMetrics{},
		pagerecall.Config{Deadline: time.Second, PollInterval: time.Millisecond, MaxInFlight: 4},
	)

	if _, err := recaller.Recall(context.Background(), "://nonsense", nil); err == nil {
		t.Fatal("expected canonicalization error")
	}
}

func TestRecallFailsWhenPlacementFails(t *testing.T) {
	recaller := pagerecall.NewRecaller(
		&recordingPlacer{err: errors.New("no stream")},
		identityResolver{},
		&fakeDisposedPages{},
		nil,
		&countingMetrics{},
		pagerecall.Config{Deadline: time.Second, PollInterval: time.Millisecond, MaxInFlight: 4},
	)

	if _, err := recaller.Recall(
		context.Background(), "https://example.com", nil,
	); err == nil {
		t.Fatal("expected placement error")
	}
}

func TestRecallGivesUpKindOnResolverError(t *testing.T) {
	recaller := pagerecall.NewRecaller(
		&recordingPlacer{},
		identityResolver{err: errors.New("kv down")},
		&fakeDisposedPages{},
		sources(markdownSource(1)),
		&countingMetrics{},
		pagerecall.Config{Deadline: time.Second, PollInterval: time.Millisecond, MaxInFlight: 4},
	)

	result, err := recaller.Recall(
		context.Background(), "https://example.com", []pagerecall.Kind{kindMarkdown},
	)
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(result.Unavailable) != 1 {
		t.Fatalf("result = %+v", result)
	}
}

func TestRecallReturnsUnavailableWhenDisposedBeforeDeadline(t *testing.T) {
	disposed := &steppingDisposedPages{revisionAt: 2}
	recaller := pagerecall.NewRecaller(
		&recordingPlacer{},
		identityResolver{},
		disposed,
		sources(markdownSource(1_000_000)),
		&countingMetrics{},
		pagerecall.Config{Deadline: time.Second, PollInterval: time.Millisecond, MaxInFlight: 4},
	)

	start := time.Now()
	result, err := recaller.Recall(
		context.Background(), "https://example.com", []pagerecall.Kind{kindMarkdown},
	)
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(result.Unavailable) != 1 {
		t.Fatalf("result = %+v", result)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("recall did not return ahead of the deadline, took %v", elapsed)
	}
}

func TestRecallIgnoresDisposalAtBaselineRevision(t *testing.T) {
	disposed := &fakeDisposedPages{revision: 5}
	recaller := pagerecall.NewRecaller(
		&recordingPlacer{},
		identityResolver{},
		disposed,
		sources(markdownSource(1_000_000)),
		&countingMetrics{},
		pagerecall.Config{
			Deadline:     20 * time.Millisecond,
			PollInterval: time.Millisecond,
			MaxInFlight:  4,
		},
	)

	result, err := recaller.Recall(
		context.Background(), "https://example.com", []pagerecall.Kind{kindMarkdown},
	)
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(result.Unavailable) != 1 {
		t.Fatalf("result = %+v", result)
	}
}

func TestRecallFailsWhenDisposalBaselineReadFails(t *testing.T) {
	recaller := pagerecall.NewRecaller(
		&recordingPlacer{},
		identityResolver{},
		&fakeDisposedPages{err: errors.New("kv down")},
		sources(markdownSource(1)),
		&countingMetrics{},
		pagerecall.Config{Deadline: time.Second, PollInterval: time.Millisecond, MaxInFlight: 4},
	)

	if _, err := recaller.Recall(
		context.Background(), "https://example.com", []pagerecall.Kind{kindMarkdown},
	); err == nil {
		t.Fatal("expected baseline read error")
	}
}

type blockingPlacer struct {
	entered chan struct{}
	release chan struct{}
}

func (p *blockingPlacer) Place(_ context.Context, _ string) error {
	close(p.entered)
	<-p.release
	return nil
}
