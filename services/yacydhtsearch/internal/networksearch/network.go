// Package networksearch ranks what the peers of the configured network hold for
// one query, inside one whole-query time budget.
package networksearch

import (
	"context"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peersearch"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchresult"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacyproto"
)

type PeerSelection interface {
	PeersFor(
		ctx context.Context,
		query searchquery.Query,
		askablePeers []peerdirectory.AskablePeer,
	) []peerdirectory.AskablePeer
}

type PerformedNetworkSearch struct {
	AmountOfAskedPeers                 int
	AmountOfAnsweringPeers             int
	AmountOfPeersThatSentItems         int
	AmountOfItemsAcrossAnswers         int
	AmountOfRepeatedItemsAcrossAnswers int
	AmountOfItemsInRanking             int
	TimeSpent                          time.Duration
}

type SearchOutcome int

const (
	PeersAsked SearchOutcome = iota
	NoIndexedTermInQuery
	NoPeerToAsk
)

type NetworkSearchObserver interface {
	NetworkSearchPerformed(ctx context.Context, search PerformedNetworkSearch)
}

type Network struct {
	networkName        string
	peerDirectory      *peerdirectory.Directory
	peerSelection      PeerSelection
	peerSearch         peersearch.Peers
	queryBudget        time.Duration
	peerCallBudget     time.Duration
	peerItemsCeiling   int
	rankedItemsCeiling int
	ringPartitions     yacymodel.DHTRingPartitions
	observer           NetworkSearchObserver
}

//nolint:revive // argument-limit: nine explicit, independently-meaningful collaborators
func New(
	networkName string,
	peerDirectory *peerdirectory.Directory,
	peerSelection PeerSelection,
	peerSearch peersearch.Peers,
	queryBudget, peerCallBudget time.Duration,
	peerItemsCeiling, rankedItemsCeiling int,
	ringPartitions yacymodel.DHTRingPartitions,
	observer NetworkSearchObserver,
) Network {
	return Network{
		networkName:        networkName,
		peerDirectory:      peerDirectory,
		peerSelection:      peerSelection,
		peerSearch:         peerSearch,
		queryBudget:        queryBudget,
		peerCallBudget:     peerCallBudget,
		peerItemsCeiling:   peerItemsCeiling,
		rankedItemsCeiling: rankedItemsCeiling,
		ringPartitions:     ringPartitions,
		observer:           observer,
	}
}

func (n Network) Search(
	ctx context.Context,
	query searchquery.Query,
) (searchresult.Ranking, SearchOutcome) {
	if len(query.Terms) == 0 {
		return searchresult.Ranking{}, NoIndexedTermInQuery
	}

	ctx, stopQueryBudget := context.WithTimeout(ctx, n.queryBudget)
	defer stopQueryBudget()
	startedAt := time.Now()

	chosenPeers := n.peerSelection.PeersFor(ctx, query, n.peerDirectory.AskablePeers(ctx))
	if len(chosenPeers) == 0 {
		return searchresult.Ranking{}, NoPeerToAsk
	}
	n.peerDirectory.MarkPeersAsked(ctx, chosenPeers)

	answers := n.peerSearch.Ask(ctx, chosenPeers, n.requestFor(query))
	ranking := searchresult.RankingFrom(itemsOfEachAnswer(answers), n.rankedItemsCeiling)
	n.observer.NetworkSearchPerformed(ctx, PerformedNetworkSearch{
		AmountOfAskedPeers:                 len(chosenPeers),
		AmountOfAnsweringPeers:             len(answers),
		AmountOfPeersThatSentItems:         amountOfPeersThatSentItems(answers),
		AmountOfItemsAcrossAnswers:         amountOfItemsAcrossAnswers(answers),
		AmountOfRepeatedItemsAcrossAnswers: amountOfRepeatedItemsAcrossAnswers(answers),
		AmountOfItemsInRanking:             len(ranking.Items),
		TimeSpent:                          time.Since(startedAt),
	})

	return ranking, PeersAsked
}

func (n Network) requestFor(query searchquery.Query) yacyproto.SearchRequest {
	return yacyproto.SearchRequest{
		NetworkName: n.networkName,
		Query:       query.TermHashes(),
		Exclude:     query.ExclusionHashes(),
		Count:       n.peerItemsCeiling,
		Time:        int(n.peerCallBudget.Milliseconds()),
		Partitions:  int(n.ringPartitions),
		ContentDom:  yacyproto.ContentDomainText,
		Language:    query.Language,
	}
}

func itemsOfEachAnswer(answers []peersearch.Answer) [][]searchresult.Item {
	items := make([][]searchresult.Item, 0, len(answers))
	for _, answer := range answers {
		items = append(items, answer.Items)
	}

	return items
}

func amountOfPeersThatSentItems(answers []peersearch.Answer) int {
	var peersThatSentItems int
	for _, answer := range answers {
		if len(answer.Items) == 0 {
			continue
		}
		peersThatSentItems++
	}

	return peersThatSentItems
}

func amountOfItemsAcrossAnswers(answers []peersearch.Answer) int {
	var answeredItems int
	for _, answer := range answers {
		answeredItems += len(answer.Items)
	}

	return answeredItems
}

func amountOfRepeatedItemsAcrossAnswers(answers []peersearch.Answer) int {
	answeredAddresses := make(map[yacymodel.URLHash]struct{}, amountOfItemsAcrossAnswers(answers))
	var repeatedItems int
	for _, answer := range answers {
		for _, item := range answer.Items {
			if _, answeredBefore := answeredAddresses[item.Hash]; answeredBefore {
				repeatedItems++

				continue
			}
			answeredAddresses[item.Hash] = struct{}{}
		}
	}

	return repeatedItems
}

type NetworkSearchObservers []NetworkSearchObserver

func (observers NetworkSearchObservers) NetworkSearchPerformed(
	ctx context.Context,
	search PerformedNetworkSearch,
) {
	for _, observer := range observers {
		observer.NetworkSearchPerformed(ctx, search)
	}
}
