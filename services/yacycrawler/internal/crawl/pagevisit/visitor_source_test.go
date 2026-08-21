package pagevisit_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pageabsorption"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pagevisit"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/reachedpagepublication"
)

type recordingAbsorberSource struct {
	absorber         *fakeAbsorption
	indexingRefusals []pageabsorption.IndexingRefusal
}

func (s *recordingAbsorberSource) AbsorberFor(
	indexingRefusal pageabsorption.IndexingRefusal,
) pageabsorption.Absorber {
	s.indexingRefusals = append(s.indexingRefusals, indexingRefusal)
	return s.absorber
}

func TestVisitorForTakesItsAbsorberFromTheIndexingRefusal(t *testing.T) {
	absorbers := &recordingAbsorberSource{absorber: &fakeAbsorption{
		links: map[string][]string{"http://host/": {"http://host/next"}},
	}}
	observer := newObserver()
	source := pagevisit.New(
		fetchOf(fetchedOutcome()),
		&fakeRecrawl{due: true},
		absorbers,
		observer,
		reachedpagepublication.NewPublisher(observer, &fakeReachedPages{}),
	)

	visitor := source.VisitorFor(pageabsorption.Ignored)

	if len(absorbers.indexingRefusals) != 1 ||
		absorbers.indexingRefusals[0] != pageabsorption.Ignored {
		t.Fatalf("absorber asked for %v, want one ignored refusal", absorbers.indexingRefusals)
	}
	outcome := visitHost(t, visitor)
	if len(outcome.DiscoveredURLs) != 1 || outcome.DiscoveredURLs[0] != "http://host/next" {
		t.Fatalf("the visitor does not absorb through its source, got %+v", outcome)
	}
}
