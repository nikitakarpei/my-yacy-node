package judgedqueries_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pagefetchershttp "github.com/nikitakarpei/yacy-rwi-node/pagefetch/pagefetchers/http"
	"github.com/nikitakarpei/yacy-rwi-node/pageformats"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/peerorder"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/itemsordering/relevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/pagereading"
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
	recordingSwitch    = "YACYDHTSEARCH_RECORD_JUDGED_QUERIES"
	networkName        = "freeworld"
	partitionExponent  = 4
	maxResponseBytes   = 4 * 1024 * 1024
	directoryCapacity  = 4096
	peerChoiceCooldown = 5 * time.Second
	refreshInterval    = 5 * time.Minute
	probeBudget        = 3 * time.Second
	probesInFlight     = 24
	peerCallsPerQuery  = 24
	peerItemsCeiling   = 10
	rankedItemsCeiling = 50
	queryBudget        = 15 * time.Second

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
	"recette crêpes",
	"programación funcional",
	"programmazione funzionale",
	"сборка ядра linux",
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
	reading := pageReadingOverTheWeb(t)
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
		maxResponseBytes,
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
			peerCallsPerQuery,
			wordjoined.WordJoinedSpreadObservers{},
		),
		peermatched.New(
			peers,
			choice,
			peerItemsCeiling,
			peerCallsPerQuery,
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

func pageReadingOverTheWeb(t *testing.T) pagereading.Reading {
	t.Helper()

	formatDerivations, err := pageformats.New()
	if err != nil {
		t.Fatalf("page format derivations: %v", err)
	}

	return pagereading.New(
		pagefetchershttp.New(
			nil,
			pagefetchershttp.ProxyDialTunnel,
			pageFetchUserAgent,
			pageByteCeiling,
			pageReadBudget,
		),
		formatDerivations,
		pageReadBudget,
		snippetLengthCeiling,
		silentPageReadingObserver{},
	)
}

func recordOneJudgedQuery(
	t *testing.T,
	spread querySpread,
	reading pagereading.Reading,
	directory *peerdirectory.Directory,
	query string,
) {
	t.Helper()

	answers := answersCarryingThePageTextOfEachDocument(
		t, reading, query, answersOfOneQuery(t, spread, directory, query),
	)
	writeFixtureFile(t, recordedAnswersFileOf(query), recordedAnswersOf(query, answers))
	pooledDocuments := pooledDocumentsOf(answers)
	writeFixtureFile(t, queryJudgmentsFileOf(query), queryJudgmentsOfThePool(
		query,
		pooledDocuments,
		queryJudgmentsInTheFile(t, queryJudgmentsFileOf(query)),
	))
	t.Logf("%q answered %d documents, %d of them pooled",
		query, len(answers.ItemOfEachAnsweredDocument()), len(pooledDocuments))
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
		ctx, searchquery.QueryFrom(query), directory.AskablePeers(ctx),
	)
}

func answersCarryingThePageTextOfEachDocument(
	t *testing.T,
	reading pagereading.Reading,
	query string,
	answers peeranswers.AnsweredQuery,
) peeranswers.AnsweredQuery {
	t.Helper()

	candidates := relevance.Ordering{}.OrderedItemsOf(answers)
	pageTextPerDocument := reading.PageTextPerDocument(
		t.Context(),
		searchquery.QueryFrom(query).TermHashes(),
		pagesToReadOf(candidates[:min(pagesReadPerQuery, len(candidates))]),
	)

	return answers.CarryingTheTextOfEachDocument(
		documentTextPerDocumentOf(pageTextPerDocument),
	)
}

func pagesToReadOf(candidates []peeranswers.AnsweredItem) []pagereading.PageToRead {
	pagesToRead := make([]pagereading.PageToRead, 0, len(candidates))
	for _, candidate := range candidates {
		pagesToRead = append(pagesToRead, pagereading.PageToRead{
			Document: candidate.Metadata.Hash,
			Address:  candidate.Metadata.Address,
		})
	}

	return pagesToRead
}

func documentTextPerDocumentOf(
	pageTextPerDocument map[yacymodel.URLHash]pagereading.PageText,
) map[yacymodel.URLHash]peeranswers.DocumentText {
	documentTextPerDocument := make(
		map[yacymodel.URLHash]peeranswers.DocumentText, len(pageTextPerDocument),
	)
	for document, pageText := range pageTextPerDocument {
		documentTextPerDocument[document] = peeranswers.DocumentText{
			HitsPerQueryWord: pageText.HitsPerQueryWord,
			AmountOfWords:    pageText.AmountOfWords,
			Snippet:          pageText.Snippet,
		}
	}

	return documentTextPerDocument
}

func pooledDocumentsOf(answers peeranswers.AnsweredQuery) []judgedDocument {
	var pooledDocuments []judgedDocument
	pooled := map[yacymodel.URLHash]struct{}{}
	for _, orderedItems := range [][]peeranswers.AnsweredItem{
		relevance.Ordering{}.OrderedItemsOf(answers),
		peerorder.Ordering{}.OrderedItemsOf(answers),
	} {
		for _, item := range orderedItems[:min(judgedItemsCeiling, len(orderedItems))] {
			if _, alreadyPooled := pooled[item.Metadata.Hash]; alreadyPooled {
				continue
			}
			pooled[item.Metadata.Hash] = struct{}{}
			pooledDocuments = append(pooledDocuments, judgedDocument{
				Hash:    item.Metadata.Hash,
				Address: item.Metadata.Address,
				Title:   item.Metadata.Title,
			})
		}
	}

	return pooledDocuments
}

func recordedAnswersFileOf(query string) string {
	return filepath.Join(recordedAnswersDirectory, fileNameOf(query))
}

func queryJudgmentsFileOf(query string) string {
	return filepath.Join(queryJudgmentsDirectory, fileNameOf(query))
}

func fileNameOf(query string) string {
	return strings.Join(strings.Fields(strings.ToLower(query)), "-") + ".json"
}
