package judgedqueries_test

import (
	"context"
	"math/rand/v2"
	"net/http"
	"testing"
	"time"

	hedgedelaysconstant "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/hedgedelays/constant"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peercallwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectoryrefresh"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerlivenesswire"
	peerpresencememory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerpresence/memory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerreliability"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/presenceaccrual"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/bywordcount"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/peermatched"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/replicaasks"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/stalepeersources/leastreliable"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/yacyseedlist"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	networkName                    = "freeworld"
	partitionExponent              = 4
	maxResponseBytes               = 4 * 1024 * 1024
	directoryCapacity              = 4096
	refreshInterval                = 5 * time.Minute
	probeBudget                    = 3 * time.Second
	probesInFlight                 = 24
	networkRedundancy              = 3
	peerCallsInFlight              = 48
	urlMetadataCallBudget          = 5 * time.Second
	searchCallBudget               = 5 * time.Second
	peerItemsCeiling               = 10
	compoundWordsCeiling           = 4
	urlMetadataAskDocumentsCeiling = 1000
	documentsToMatchCeiling        = 1000
	queryBudget                    = 15 * time.Second
)

var seedlistURLs = []string{
	"http://sixcooler.de/yacy/seed.txt",
	"https://sonst.mifritscher.de/yacy/seed.txt",
	"http://5.45.105.16/yacyseed",
	"http://yacy.v16.de/seed/seed.txt",
	"https://frank-siebert.de/seed.txt",
	"http://seedlist.wertewesten.net/seed.txt",
}

type peersOfTheNetwork struct {
	directory   *peerdirectory.Directory
	reliability peerreliability.Reliability
}

func peersOfTheNetworkRefreshedOnce(t *testing.T) peersOfTheNetwork {
	t.Helper()

	presence := peerpresencememory.New(
		presenceaccrual.PresenceAccrualLimits{
			Capacity:        directoryCapacity,
			ContinuityLimit: refreshInterval,
		},
		presenceaccrual.PresenceAccrualObservers{},
	)
	reliability := peerreliability.New(
		presence, peerreliability.DefaultReliabilityWeights(), time.Now,
	)
	directory := peerdirectory.New(
		peerdirectory.DirectoryLimits{Capacity: directoryCapacity},
		time.Now,
		leastreliable.New(reliability, refreshInterval, time.Now),
		peerdirectory.DirectoryObservers{presence},
	)
	peerdirectoryrefresh.New(
		yacyseedlist.New(
			http.DefaultClient,
			seedlistURLs,
			maxResponseBytes,
			silentSeedlistObserver{},
		),
		directory,
		peerlivenesswire.New(
			http.DefaultClient, networkName, peerlivenesswire.PeerLivenessObservers{},
		),
		presence,
		peerdirectoryrefresh.ProbeLimits{
			ProbeBudget:    probeBudget,
			ProbesInFlight: probesInFlight,
		},
	).RefreshOnce(t.Context())

	return peersOfTheNetwork{directory: directory, reliability: reliability}
}

type silentSeedlistObserver struct{}

func (silentSeedlistObserver) SeedlistRead(context.Context, string, int)          {}
func (silentSeedlistObserver) SeedlistUnreachable(context.Context, string, error) {}
func (silentSeedlistObserver) SeedlistUnreadable(context.Context, string, error)  {}

type querySpread interface {
	SpreadOverPeers(
		ctx context.Context,
		query searchquery.Query,
		askablePeers []peerdirectory.AskablePeer,
	) queryanswers.AnsweredQuery
}

func (peers peersOfTheNetwork) querySpread(t *testing.T) querySpread {
	t.Helper()

	partitions := ringPartitions(t)
	calledPeers := peercallwire.New(
		http.DefaultClient,
		peercallwire.SearchedNetwork{Name: networkName, RingPartitions: partitions},
		peercallwire.PeerCallLimits{
			MaxResponseBytes:      maxResponseBytes,
			PeerCallsInFlight:     peerCallsInFlight,
			URLMetadataCallBudget: urlMetadataCallBudget,
			SearchCallBudget:      searchCallBudget,
		},
		peercallwire.PeerCallObservers{},
	)
	everyReplica := replicaasks.New(
		calledPeers,
		hedgedelaysconstant.New(searchCallBudget),
		networkRedundancy,
		replicaasks.ReplicaAsksObservers{},
	)

	return spreadChoosingPeers{
		peerChoice: peerchoice.New(
			partitions, networkRedundancy, peers.reliability, peerchoice.PeerChoiceObservers{},
		),
		byWordCount: bywordcount.New(
			wordjoined.New(
				everyReplica,
				calledPeers,
				noRememberedQueryWordDocumentAmounts{},
				wordjoined.URLMetadataLookupCutoff{},
				rand.UintN,
				urlMetadataAskDocumentsCeiling,
				documentsToMatchCeiling,
				peerItemsCeiling,
				partitions,
				yacymodel.PeersHoldingOneWordOf(partitions, networkRedundancy),
				wordjoined.WordJoinedSpreadObservers{},
			),
			peermatched.New(
				everyReplica,
				peerItemsCeiling,
				peermatched.PeerMatchedSpreadObservers{},
			),
		),
	}
}

func ringPartitions(t *testing.T) yacymodel.DHTRingPartitions {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(partitionExponent)
	if err != nil {
		t.Fatalf("partitions from exponent %d: %v", partitionExponent, err)
	}

	return partitions
}

type spreadChoosingPeers struct {
	peerChoice  peerchoice.Choice
	byWordCount bywordcount.Spread
}

func (spread spreadChoosingPeers) SpreadOverPeers(
	ctx context.Context,
	query searchquery.Query,
	askablePeers []peerdirectory.AskablePeer,
) queryanswers.AnsweredQuery {
	chosenPeersPerQueryWord := spread.peerChoice.ChosenPeersPerQueryWordFor(
		ctx, query.HashesOfWordsAndCompoundWordsUpTo(compoundWordsCeiling), askablePeers,
	)

	return spread.byWordCount.SpreadOverPeers(ctx, query, chosenPeersPerQueryWord)
}

type noRememberedQueryWordDocumentAmounts struct{}

func (noRememberedQueryWordDocumentAmounts) DocumentAmountsOf(
	context.Context,
	[]yacymodel.Hash,
) map[yacymodel.Hash]int {
	return nil
}

func (noRememberedQueryWordDocumentAmounts) Remember(context.Context, map[yacymodel.Hash]int) {}
