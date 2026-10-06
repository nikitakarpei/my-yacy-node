package networksearch_test

import (
	"context"
	"slices"
	"sync"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/networksearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const grownAddress = "https://grown.example/"

type spreadGrowingOnce struct {
	grown     queryfindings.Findings
	readAhead <-chan struct{}
}

func (s spreadGrowingOnce) SpreadOverPeers(
	_ context.Context,
	_ searchquery.Query,
	_ peerchoice.ChosenPeersPerQueryWord,
	growth queryfindings.Growth,
) queryfindings.Findings {
	growth.FindingsGrew(s.grown)
	<-s.readAhead

	return s.grown
}

type pagesRecordingWhatIsReadAhead struct {
	mutex            sync.Mutex
	addressesAhead   []string
	firstReadAhead   chan struct{}
	firstReadAheadOK sync.Once
}

func newPagesRecordingWhatIsReadAhead() *pagesRecordingWhatIsReadAhead {
	return &pagesRecordingWhatIsReadAhead{firstReadAhead: make(chan struct{})}
}

func (p *pagesRecordingWhatIsReadAhead) Start(_ []yacymodel.Hash) networksearch.PageReadingRun {
	return p
}

func (p *pagesRecordingWhatIsReadAhead) ReadAhead(
	_ context.Context,
	pagesToRead []pagereading.PageToRead,
) {
	p.mutex.Lock()
	for _, page := range pagesToRead {
		p.addressesAhead = append(p.addressesAhead, page.Address)
	}
	p.mutex.Unlock()
	p.firstReadAheadOK.Do(func() { close(p.firstReadAhead) })
}

func (*pagesRecordingWhatIsReadAhead) Read(
	_ context.Context,
	_ []pagereading.PageToRead,
) pagereading.PagesRead {
	return pagereading.PagesRead{}
}

func (*pagesRecordingWhatIsReadAhead) Finish(_ context.Context) {}

func (p *pagesRecordingWhatIsReadAhead) readAhead() []string {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return slices.Clone(p.addressesAhead)
}

func networkReadingFrom(
	t *testing.T,
	pages *pagesRecordingWhatIsReadAhead,
) networksearch.Network {
	t.Helper()

	return networksearch.New(
		directoryAnsweringAt(t, peerHolding(t)),
		everyAskablePeer{},
		spreadGrowingOnce{
			grown: queryfindings.Findings{
				QueryWords: []yacymodel.Hash{yacymodel.WordHash("berlin")},
				FoundDocuments: []queryfindings.FoundDocument{{
					Hash:    documentOf(t, grownAddress),
					Address: grownAddress,
					Facts:   oneHitOfTheWord("berlin"),
				}},
			},
			readAhead: pages.firstReadAhead,
		},
		pages,
		orderingInTheFoundOrder{},
		queryBudget,
		pageReadBudget,
		pagesReadPerQuery,
		pagesReadPerSite,
		recordCeiling,
		compoundWordsCeiling,
		networksearch.NetworkSearchObservers{&recordedQuery{}},
	)
}

func TestASearchReadingPagesAheadReadsThePagesOfTheFindingsAsTheyGrow(t *testing.T) {
	t.Parallel()

	pages := newPagesRecordingWhatIsReadAhead()
	network := networkReadingFrom(t, pages)

	network.Search(t.Context(), queryreading.QueryFrom("berlin", ""))

	if got := pages.readAhead(); !slices.Equal(got, []string{grownAddress}) {
		t.Fatalf("the pages read ahead are %v, want the page of the grown findings", got)
	}
}
