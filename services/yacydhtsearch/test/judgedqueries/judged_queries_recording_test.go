package judgedqueries_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/relevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peeranswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peercallwire"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerchoice"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectoryrefresh"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerlivenesswire"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerselections/dhtdistance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/bywordcount"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/peermatched"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryspreads/wordjoined"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/stalepeersources/leastrecentlyanswered"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/yacyseedlist"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	recordingSwitch     = "YACYDHTSEARCH_RECORD_JUDGED_QUERIES"
	networkName         = "freeworld"
	partitionExponent   = 4
	maxResponseBytes    = 4 * 1024 * 1024
	directoryCapacity   = 4096
	peerChoiceCooldown  = 5 * time.Second
	refreshInterval     = 5 * time.Minute
	probeBudget         = 3 * time.Second
	probesInFlight      = 24
	peersHoldingOneWord = 48
	peerCallsInFlight   = 48
	peerCallBudget      = 5 * time.Second
	peerItemsCeiling    = 10
	rankedItemsCeiling  = 50
	queryBudget         = 15 * time.Second

	pagesReadPerQuery    = 50
	pageReadBudget       = 10 * time.Second
	pageByteCeiling      = 4 * 1024 * 1024
	snippetLengthCeiling = 300
	pageFetchUserAgent   = "yacydhtsearch (+https://yacy.net)"
)

var seedlistURLs = []string{
	"http://sixcooler.de/yacy/seed.txt",
	"https://sonst.mifritscher.de/yacy/seed.txt",
	"http://5.45.105.16/yacyseed",
	"http://yacy.v16.de/seed/seed.txt",
	"https://frank-siebert.de/seed.txt",
	"http://seedlist.wertewesten.net/seed.txt",
}

var judgedQueries = []string{
	"wikipedia",
	"debian",
	"kubernetes",
	"bundestag",
	"rust",
	"photosynthesis",
	"openstreetmap",
	"heise",
	"fahrrad",
	"mastodon",
	"mercury",
	"python",
	"jaguar",
	"apple",
	"golang concurrency",
	"python asyncio",
	"linux kernel",
	"arch wiki",
	"berlin wetter",
	"climate change",
	"docker compose",
	"nextcloud installation",
	"raspberry pi",
	"git rebase",
	"how to install debian",
	"how do i reset my router",
	"why does my laptop battery drain fast",
	"how to fix a slow computer",
	"what is the best way to learn a new language",
	"how to remove a stripped screw",
	"why do cats knead blankets",
	"how to negotiate a salary raise",
	"what to do when locked out of your house",
	"free software foundation",
	"open source search engine",
	"deutsche bahn fahrplan",
	"static site generator",
	"tor browser download",
	"wolfgang amadeus mozart",
	"machine learning tutorial",
	"yacy peer to peer search",
	"self hosted email server",
	"gnu emacs manual",
	"chaos computer club",
	"apache httpd",
	"postgresql documentation",
	"libreoffice",
	"fahrradwerkstatt berlin",
	"how does public key cryptography work",
	"difference between tcp and udp",
	"best practices for database indexing",
	"how does dns resolution work",
	"why is my wifi connection slow",
	"what causes climate change",
	"how do vaccines work",
	"advantages of renewable energy sources",
	"how does a search engine rank pages",
	"what is the difference between http and https",
	"recette crêpes",
	"programación funcional",
	"programmazione funzionale",
	"сборка ядра linux",
	"wie funktioniert ein elektroauto",
	"berliner mauer geschichte",
	"comment apprendre le français",
	"recette tarte tatin",
	"como funciona internet",
	"mejores playas de españa",
	"storia antica di roma",
	"как приготовить борщ",
	"xqzvptl kwrmbz",
	"zzyzx quanternion glomph",
}

type querySpread interface {
	SpreadOverPeers(
		ctx context.Context,
		query searchquery.Query,
		askablePeers []peerdirectory.AskablePeer,
	) peeranswers.AnsweredQuery
}

func TestRecordWhatThePeersAnswerForTheJudgedQueries(t *testing.T) {
	if os.Getenv(recordingSwitch) == "" {
		t.Skipf("set %s to record what the peers answer", recordingSwitch)
	}

	directory := directoryOfTheNetwork(t)
	spread := querySpreadOverThePeers(t, directory)
	reading := pageTextReadingOverTheWeb(t)
	t.Logf(
		"the directory knows %d peers and can ask %d",
		len(directory.KnownPeers(t.Context())),
		len(directory.AskablePeers(t.Context())),
	)
	for _, query := range judgedQueries {
		recordOneJudgedQuery(t, spread, reading, directory, query)
	}
}

func directoryOfTheNetwork(t *testing.T) *peerdirectory.Directory {
	t.Helper()

	directory := peerdirectory.New(
		directoryCapacity,
		peerChoiceCooldown,
		time.Now,
		leastrecentlyanswered.New(),
		peerdirectory.DirectoryObservers{},
	)
	peerdirectoryrefresh.New(
		yacyseedlist.New(
			http.DefaultClient,
			seedlistURLs,
			maxResponseBytes,
			silentSeedlistObserver{},
		),
		directory,
		peerlivenesswire.New(http.DefaultClient, networkName),
		refreshInterval,
		probeBudget,
		probesInFlight,
	).RefreshOnce(t.Context())

	return directory
}

func querySpreadOverThePeers(t *testing.T, directory *peerdirectory.Directory) querySpread {
	t.Helper()

	partitions := ringPartitions(t)
	peers := peercallwire.New(
		http.DefaultClient,
		peercallwire.SearchedNetwork{Name: networkName, RingPartitions: partitions},
		peercallwire.PeerCallLimits{
			MaxResponseBytes:  maxResponseBytes,
			PeerCallsInFlight: peerCallsInFlight,
			PeerCallBudget:    peerCallBudget,
		},
		peercallwire.PeerCallObservers{},
	)
	choice := peerchoice.New(
		dhtdistance.New(partitions, dhtdistance.DHTDistanceObservers{}),
		directory,
	)

	return bywordcount.New(
		wordjoined.New(
			peers,
			choice,
			rankedItemsCeiling,
			peerItemsCeiling,
			peersHoldingOneWord,
			wordjoined.WordJoinedSpreadObservers{},
		),
		peermatched.New(
			peers,
			choice,
			peerItemsCeiling,
			peersHoldingOneWord,
			peermatched.PeerMatchedSpreadObservers{},
		),
	)
}

func ringPartitions(t *testing.T) yacymodel.DHTRingPartitions {
	t.Helper()

	partitions, err := yacymodel.DHTRingPartitionsFromExponent(partitionExponent)
	if err != nil {
		t.Fatalf("partitions from exponent %d: %v", partitionExponent, err)
	}

	return partitions
}

func recordOneJudgedQuery(
	t *testing.T,
	spread querySpread,
	reading pageTextReading,
	directory *peerdirectory.Directory,
	query string,
) {
	t.Helper()

	answers := answersOfOneQuery(t, spread, directory, query)
	pageTextPerDocument := pageTextOfTheFirstAnsweredDocuments(t, reading, answers)
	storePageTextOfTheQuery(t, query, pageTextPerDocument)
	answersCarryingThePageText := answersCarryingThePageTextOfEachDocument(
		query, answers, pageTextPerDocument,
	)
	writeFixtureFile(
		t, recordedAnswersFileOf(query), recordedAnswersOf(query, answersCarryingThePageText),
	)
	judgments := queryJudgmentsOfTheDocumentsToJudge(
		query,
		answersCarryingThePageText,
		pageTextPerDocument,
		queryJudgmentsInTheFile(t, queryJudgmentsFileOf(query)),
	)
	writeFixtureFile(t, queryJudgmentsFileOf(query), judgments)
	t.Logf("%q read the page of %d documents, judges %d, and waits for %d grades",
		query,
		len(pageTextPerDocument),
		len(judgments.JudgedDocuments),
		judgments.amountOfUngradedDocuments(),
	)
}

func answersOfOneQuery(
	t *testing.T,
	spread querySpread,
	directory *peerdirectory.Directory,
	query string,
) peeranswers.AnsweredQuery {
	t.Helper()

	ctx, stopQueryBudget := context.WithTimeout(t.Context(), queryBudget)
	defer stopQueryBudget()

	return spread.SpreadOverPeers(
		ctx, searchquery.QueryFrom(query, ""), directory.AskablePeers(ctx),
	)
}

func pageTextOfTheFirstAnsweredDocuments(
	t *testing.T,
	reading pageTextReading,
	answers peeranswers.AnsweredQuery,
) map[yacymodel.URLHash]string {
	t.Helper()

	candidates := relevance.New(relevance.DefaultScoreWeights()).OrderedItemsOf(answers)

	return reading.pageTextPerDocument(
		t.Context(), candidates[:min(pagesReadPerQuery, len(candidates))],
	)
}

func recordedAnswersFileOf(query string) string {
	return filepath.Join(recordedAnswersDirectory, queryInFileNames(query)+".json")
}

func queryJudgmentsFileOf(query string) string {
	return filepath.Join(queryJudgmentsDirectory, queryInFileNames(query)+".json")
}

func queryInFileNames(query string) string {
	return strings.Join(strings.Fields(strings.ToLower(query)), "-")
}
