package pagepublication

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/retrydelay"
)

type fakeRepresentation struct {
	kind          yacycrawlcontract.PageRepresentationKind
	contentFormat contentformatgraph.Format
	mu            sync.Mutex
	published     []string
	failWith      error
}

func (o *fakeRepresentation) Kind() yacycrawlcontract.PageRepresentationKind {
	return o.kind
}

func (o *fakeRepresentation) ContentFormat() contentformatgraph.Format {
	if o.contentFormat == "" {
		return contentformatgraph.FormatReadableText
	}
	return o.contentFormat
}

func (o *fakeRepresentation) Frame(page Page, _ []byte) ([][]byte, error) {
	return [][]byte{[]byte(page.CanonicalURL)}, nil
}

func (o *fakeRepresentation) Publish(_ context.Context, messages [][]byte) error {
	if o.failWith != nil {
		return o.failWith
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.published = append(o.published, string(messages[0]))
	return nil
}

type flakyRepresentation struct {
	failuresLeft int
	published    int
}

func (*flakyRepresentation) Kind() yacycrawlcontract.PageRepresentationKind {
	return yacycrawlcontract.PageRepresentationKindRWI
}

func (*flakyRepresentation) ContentFormat() contentformatgraph.Format {
	return contentformatgraph.FormatReadableText
}

func (*flakyRepresentation) Frame(Page, []byte) ([][]byte, error) {
	return [][]byte{}, nil
}

func (o *flakyRepresentation) Publish(context.Context, [][]byte) error {
	if o.failuresLeft > 0 {
		o.failuresLeft--
		return TransientPublicationError{Err: errors.New("stream full")}
	}
	o.published++
	return nil
}

type fakeDerivation struct {
	sourceFormat contentformatgraph.Format
	targetFormat contentformatgraph.Format
}

func (d fakeDerivation) SourceFormat() contentformatgraph.Format { return d.sourceFormat }
func (d fakeDerivation) TargetFormat() contentformatgraph.Format { return d.targetFormat }

func (fakeDerivation) Derive(_ string, body []byte) ([]byte, error) { return body, nil }

func derivations() []contentformatgraph.Derivation {
	return []contentformatgraph.Derivation{
		fakeDerivation{
			sourceFormat: contentformatgraph.FormatDocumentHTML,
			targetFormat: contentformatgraph.FormatReadableText,
		},
		fakeDerivation{
			sourceFormat: contentformatgraph.FormatReadableText,
			targetFormat: contentformatgraph.FormatReadableText,
		},
		fakeDerivation{
			sourceFormat: contentformatgraph.FormatDocumentHTML,
			targetFormat: contentformatgraph.FormatMarkdown,
		},
	}
}

type recordingObserver struct {
	mu        sync.Mutex
	published map[yacycrawlcontract.PageRepresentationKind]int
	waits     int
}

func newObserver() *recordingObserver {
	return &recordingObserver{
		published: map[yacycrawlcontract.PageRepresentationKind]int{},
	}
}

func (o *recordingObserver) PagePublished(
	representation yacycrawlcontract.PageRepresentationKind,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.published[representation]++
}

func (o *recordingObserver) PublicationWaited() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.waits++
}

type manualClock struct{ now time.Time }

func (c *manualClock) Now() time.Time { return c.now }

func (c *manualClock) Sleep(ctx context.Context, duration time.Duration) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("manual clock: %w", err)
	}
	c.now = c.now.Add(duration)
	return nil
}

func newPublisher(representations []PageRepresentation, observer PublicationProgress) *Publisher {
	return New(
		contentformatgraph.New(derivations()),
		representations,
		observer,
		&manualClock{},
		retrydelay.Bounds{Floor: time.Millisecond, Ceiling: time.Millisecond},
	)
}

func readablePage() Page {
	return Page{
		CanonicalURL: "http://host/",
		Body:         []byte("body"),
		Format:       contentformatgraph.FormatReadableText,
	}
}

func TestPublishReachesEveryRepresentation(t *testing.T) {
	rwi := &fakeRepresentation{kind: yacycrawlcontract.PageRepresentationKindRWI}
	text := &fakeRepresentation{kind: yacycrawlcontract.PageRepresentationKindText}
	p := newPublisher([]PageRepresentation{rwi, text}, newObserver())

	if err := p.Publish(t.Context(), readablePage()); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if len(rwi.published) != 1 || len(text.published) != 1 {
		t.Fatalf("representations not both advanced: rwi=%v text=%v", rwi.published, text.published)
	}
}

func TestPublishFailsWhenAnyRepresentationCannotDerive(t *testing.T) {
	rwi := &fakeRepresentation{kind: yacycrawlcontract.PageRepresentationKindRWI}
	markdown := &fakeRepresentation{
		kind:          yacycrawlcontract.PageRepresentationKindMarkdown,
		contentFormat: contentformatgraph.FormatMarkdown,
	}
	observer := newObserver()
	p := newPublisher([]PageRepresentation{rwi, markdown}, observer)

	if err := p.Publish(
		t.Context(),
		readablePage(),
	); !errors.Is(
		err,
		ErrRepresentationUnresolvable,
	) {
		t.Fatalf("want ErrRepresentationUnresolvable, got %v", err)
	}
	if len(rwi.published) != 0 {
		t.Fatalf(
			"representation published despite a sibling failing to derive: rwi=%v",
			rwi.published,
		)
	}
	if len(markdown.published) != 0 {
		t.Fatalf("refusing representation advanced: markdown=%v", markdown.published)
	}
}

func TestPublishFailsWhenNoRepresentationDerives(t *testing.T) {
	rwi := &fakeRepresentation{
		kind:          yacycrawlcontract.PageRepresentationKindRWI,
		contentFormat: contentformatgraph.FormatMarkdown,
	}
	p := newPublisher([]PageRepresentation{rwi}, newObserver())

	if err := p.Publish(
		t.Context(),
		readablePage(),
	); !errors.Is(
		err,
		ErrRepresentationUnresolvable,
	) {
		t.Fatalf("want ErrRepresentationUnresolvable, got %v", err)
	}
	if len(rwi.published) != 0 {
		t.Fatalf("refusing representation advanced: rwi=%v", rwi.published)
	}
}

func TestPublishHardErrorFails(t *testing.T) {
	representation := &fakeRepresentation{
		kind:     yacycrawlcontract.PageRepresentationKindRWI,
		failWith: errors.New("hard broker error"),
	}
	p := newPublisher([]PageRepresentation{representation}, newObserver())

	if err := p.Publish(t.Context(), readablePage()); err == nil {
		t.Fatal("hard publish error should fail publication")
	}
}

func TestPublishRetriesTransientFailure(t *testing.T) {
	representation := &flakyRepresentation{failuresLeft: 2}
	observer := newObserver()
	p := newPublisher([]PageRepresentation{representation}, observer)

	if err := p.Publish(t.Context(), readablePage()); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if representation.published != 1 {
		t.Fatalf(
			"transient publish should retry then succeed: published=%d",
			representation.published,
		)
	}
	if observer.waits != 2 {
		t.Fatalf("want a recorded wait per transient failure, got %d", observer.waits)
	}
}
