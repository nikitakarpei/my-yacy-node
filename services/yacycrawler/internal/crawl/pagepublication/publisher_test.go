package pagepublication_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/pagescrape/contentformatgraph"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawlcontract"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagepublication"
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

func (o *fakeRepresentation) Frame(page pagepublication.Page, _ []byte) ([][]byte, error) {
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

func (*flakyRepresentation) Frame(pagepublication.Page, []byte) ([][]byte, error) {
	return [][]byte{}, nil
}

func (o *flakyRepresentation) Publish(context.Context, [][]byte) error {
	if o.failuresLeft > 0 {
		o.failuresLeft--
		return pagepublication.TransientPublicationError{Err: errors.New("stream full")}
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
	mu          sync.Mutex
	published   map[yacycrawlcontract.PageRepresentationKind]int
	underivable map[yacycrawlcontract.PageRepresentationKind]int
	waits       int
}

func newObserver() *recordingObserver {
	return &recordingObserver{
		published:   map[yacycrawlcontract.PageRepresentationKind]int{},
		underivable: map[yacycrawlcontract.PageRepresentationKind]int{},
	}
}

func (o *recordingObserver) PagePublished(
	representation yacycrawlcontract.PageRepresentationKind,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.published[representation]++
}

func (o *recordingObserver) RepresentationUnderivable(
	representation yacycrawlcontract.PageRepresentationKind,
) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.underivable[representation]++
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

func newPublisher(
	representations []pagepublication.PageRepresentation,
	observer pagepublication.PublicationProgress,
) *pagepublication.Publisher {
	return pagepublication.New(
		contentformatgraph.New(derivations()),
		representations,
		observer,
		&manualClock{},
		retrydelay.Bounds{Floor: time.Millisecond, Ceiling: time.Millisecond},
	)
}

func readablePage() pagepublication.Page {
	return pagepublication.Page{
		CanonicalURL: "http://host/",
		Body:         []byte("body"),
		Format:       contentformatgraph.FormatReadableText,
	}
}

func TestPublishReachesEveryRepresentation(t *testing.T) {
	rwi := &fakeRepresentation{kind: yacycrawlcontract.PageRepresentationKindRWI}
	text := &fakeRepresentation{kind: yacycrawlcontract.PageRepresentationKindText}
	p := newPublisher([]pagepublication.PageRepresentation{rwi, text}, newObserver())

	if err := p.Publish(t.Context(), readablePage()); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if len(rwi.published) != 1 || len(text.published) != 1 {
		t.Fatalf("representations not both advanced: rwi=%v text=%v", rwi.published, text.published)
	}
}

func TestPublishSkipsTheRepresentationThatCannotDerive(t *testing.T) {
	rwi := &fakeRepresentation{kind: yacycrawlcontract.PageRepresentationKindRWI}
	text := &fakeRepresentation{
		kind:          yacycrawlcontract.PageRepresentationKindText,
		contentFormat: contentformatgraph.FormatMarkdown,
	}
	observer := newObserver()
	p := newPublisher([]pagepublication.PageRepresentation{rwi, text}, observer)

	if err := p.Publish(t.Context(), readablePage()); err != nil {
		t.Fatalf("an underivable representation should not fail publication: %v", err)
	}
	if len(rwi.published) != 1 {
		t.Fatalf("derivable sibling withheld: rwi=%v", rwi.published)
	}
	if len(text.published) != 0 {
		t.Fatalf("underivable representation advanced: text=%v", text.published)
	}
	if observer.underivable[yacycrawlcontract.PageRepresentationKindText] != 1 {
		t.Fatalf("underivable representation unobserved: %v", observer.underivable)
	}
}

func TestPublishFailsWhenNoRepresentationDerives(t *testing.T) {
	rwi := &fakeRepresentation{
		kind:          yacycrawlcontract.PageRepresentationKindRWI,
		contentFormat: contentformatgraph.FormatMarkdown,
	}
	p := newPublisher([]pagepublication.PageRepresentation{rwi}, newObserver())

	if err := p.Publish(t.Context(), readablePage()); err == nil {
		t.Fatal("a page no representation derives should fail publication")
	}
	if len(rwi.published) != 0 {
		t.Fatalf("underivable representation advanced: rwi=%v", rwi.published)
	}
}

func TestPublishHardErrorFails(t *testing.T) {
	representation := &fakeRepresentation{
		kind:     yacycrawlcontract.PageRepresentationKindRWI,
		failWith: errors.New("hard broker error"),
	}
	p := newPublisher([]pagepublication.PageRepresentation{representation}, newObserver())

	if err := p.Publish(t.Context(), readablePage()); err == nil {
		t.Fatal("hard publish error should fail publication")
	}
}

func TestPublishRetriesTransientFailure(t *testing.T) {
	representation := &flakyRepresentation{failuresLeft: 2}
	observer := newObserver()
	p := newPublisher([]pagepublication.PageRepresentation{representation}, observer)

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
