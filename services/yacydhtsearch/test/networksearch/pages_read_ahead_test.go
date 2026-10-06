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

const (
	grownAddress      = "https://grown.example/"
	grownLaterAddress = "https://grown.example/later"
)

type spreadGrowingInTurn struct {
	findingsInTurn []queryfindings.Findings
	readAhead      <-chan struct{}
}

func (s spreadGrowingInTurn) SpreadOverPeers(
	_ context.Context,
	_ searchquery.Query,
	_ peerchoice.ChosenPeersPerQueryWord,
	growth queryfindings.Growth,
) queryfindings.Findings {
	for _, findings := range s.findingsInTurn {
		growth.FindingsGrew(findings)
		<-s.readAhead
	}

	return s.findingsInTurn[len(s.findingsInTurn)-1]
}

type pagesRecordingWhatIsReadAhead struct {
	mutex              sync.Mutex
	addressesAhead     []string
	addressesAbandoned []string
	readAhead          chan struct{}
}

func newPagesRecordingWhatIsReadAhead() *pagesRecordingWhatIsReadAhead {
	return &pagesRecordingWhatIsReadAhead{readAhead: make(chan struct{})}
}

func (p *pagesRecordingWhatIsReadAhead) Start(_ []string) networksearch.PageReadingRun {
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
	p.readAhead <- struct{}{}
}

func (p *pagesRecordingWhatIsReadAhead) Abandon(pagesToAbandon []pagereading.PageToRead) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	for _, page := range pagesToAbandon {
		p.addressesAbandoned = append(p.addressesAbandoned, page.Address)
	}
}

func (*pagesRecordingWhatIsReadAhead) Read(
	_ context.Context,
	_ []pagereading.PageToRead,
) pagereading.PagesRead {
	return pagereading.PagesRead{}
}

func (*pagesRecordingWhatIsReadAhead) Finish(_ context.Context) {}

func (p *pagesRecordingWhatIsReadAhead) readAheadAndAbandoned() ([]string, []string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return slices.Clone(p.addressesAhead), slices.Clone(p.addressesAbandoned)
}

func findingsOfTheAddress(t *testing.T, address string) queryfindings.Findings {
	t.Helper()

	return queryfindings.Findings{
		QueryWords: []yacymodel.Hash{yacymodel.WordHash("berlin")},
		FoundDocuments: []queryfindings.FoundDocument{{
			Hash:    documentOf(t, address),
			Address: address,
			Facts:   oneHitOfTheWord("berlin"),
		}},
	}
}

func networkReadingFrom(
	t *testing.T,
	pages *pagesRecordingWhatIsReadAhead,
	findingsInTurn ...queryfindings.Findings,
) networksearch.Network {
	t.Helper()

	return networksearch.New(
		directoryAnsweringAt(t, peerHolding(t)),
		everyAskablePeer{},
		spreadGrowingInTurn{findingsInTurn: findingsInTurn, readAhead: pages.readAhead},
		pages,
		orderingInTheFoundOrder{},
		queryBudget,
		pageReadBudget,
		pagesReadPerQueryCeiling,
		pagesReadPerSiteCeiling,
		recordCeiling,
		compoundWordsCeiling,
		networksearch.NetworkSearchObservers{&recordedQuery{}},
	)
}

func TestASearchReadsThePagesOfTheFindingsAheadAsTheyGrow(t *testing.T) {
	t.Parallel()

	pages := newPagesRecordingWhatIsReadAhead()
	network := networkReadingFrom(t, pages, findingsOfTheAddress(t, grownAddress))

	network.Search(t.Context(), queryreading.QueryFrom("berlin", ""))

	if readAhead, _ := pages.readAheadAndAbandoned(); !slices.Equal(
		readAhead,
		[]string{grownAddress},
	) {
		t.Fatalf("the pages read ahead are %v, want the page of the grown findings", readAhead)
	}
}

func TestASearchAbandonsAPageReadAheadThatTheGrownFindingsLeaveOut(t *testing.T) {
	t.Parallel()

	pages := newPagesRecordingWhatIsReadAhead()
	network := networkReadingFrom(
		t, pages,
		findingsOfTheAddress(t, grownAddress),
		findingsOfTheAddress(t, grownLaterAddress),
	)

	network.Search(t.Context(), queryreading.QueryFrom("berlin", ""))

	if _, abandoned := pages.readAheadAndAbandoned(); !slices.Equal(
		abandoned,
		[]string{grownAddress},
	) {
		t.Fatalf("the pages abandoned are %v, want the page the findings left out", abandoned)
	}
}
