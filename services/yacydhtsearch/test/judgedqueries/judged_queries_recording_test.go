package judgedqueries_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentrelevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/documentsordering/relevance"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

const (
	recordingSwitch = "YACYDHTSEARCH_RECORD_JUDGED_QUERIES"

	pagesReadPerQuery    = 50
	pageBudget           = 10 * time.Second
	recordingPagesBudget = 10 * time.Second
	pageByteCeiling      = 4 * 1024 * 1024
	snippetLengthCeiling = 300
	pageFetchUserAgent   = "yacydhtsearch (+https://yacy.net)"
)

var recordedQueries = []string{
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
	"payday loan",
	"viagra",
	"sildenafil",
	"online gambling",
	"weight loss",
	"minecraft download",
	"watch movies online",
	"how to make money online",
	"xqzvptl kwrmbz",
	"zzyzx quanternion glomph",
}

func TestRecordWhatThePeersAnswerForTheJudgedQueries(t *testing.T) {
	if os.Getenv(recordingSwitch) == "" {
		t.Skipf("set %s to record what the peers answer", recordingSwitch)
	}

	peers := peersOfTheNetworkRefreshedOnce(t)
	recording := judgedQueryRecording{
		spread:     peers.querySpread(t),
		fetching:   pageFetchingWithin(recordingPagesBudget),
		extraction: pageExtractionOfEveryFormat(t),
		directory:  peers.directory,
	}
	t.Logf(
		"the directory knows %d peers and can ask %d",
		len(peers.directory.KnownPeers(t.Context())),
		len(peers.directory.AskablePeers(t.Context())),
	)
	for _, query := range recordedQueries {
		t.Run(query, func(t *testing.T) {
			recording.recordOne(t, query)
		})
	}
}

type judgedQueryRecording struct {
	spread     querySpread
	fetching   pageFetching
	extraction pageExtraction
	directory  *peerdirectory.Directory
}

func (recording judgedQueryRecording) recordOne(t *testing.T, query string) {
	t.Helper()

	answers := recording.answersOf(t, query)
	pages := recording.pagesOfTheFirstDocumentsIn(t, answers)
	writeStoredPagesOf(t, query, pages)
	answersAndPageContents := answersAndPageContentsOf(
		answers,
		recording.extraction.pageContentsPerDocumentOf(
			t.Context(), answers, pagePerAddressOf(pages),
		),
	)
	writeRecordedAnswersFile(
		t, recordedAnswersFileOf(query), recordedAnswersOf(query, answersAndPageContents),
	)
	judgments := writeJudgmentsOf(t, query, answersAndPageContents)
	t.Logf("%q read the page of %d documents, judges %d, and waits for %d grades",
		query,
		len(answersAndPageContents.pageContentsPerDocument),
		len(judgments.JudgedDocuments),
		judgments.amountOfUngradedDocuments(),
	)
}

func (recording judgedQueryRecording) answersOf(
	t *testing.T, query string,
) queryanswers.AnsweredQuery {
	t.Helper()

	ctx, stopQueryBudget := context.WithTimeout(t.Context(), queryBudget)
	defer stopQueryBudget()

	return recording.spread.SpreadOverPeers(
		ctx, searchquery.QueryFrom(query, ""), recording.directory.AskablePeers(ctx),
	)
}

func (recording judgedQueryRecording) pagesOfTheFirstDocumentsIn(
	t *testing.T, answers queryanswers.AnsweredQuery,
) []storedPage {
	t.Helper()

	orderedDocuments := relevance.New(
		documentrelevance.RelevanceScorerWeighedBy(documentrelevance.DefaultRelevanceWeights()),
	).OrderedDocumentsOf(answers)

	return recording.fetching.fetchedPagesOf(
		t.Context(), orderedDocuments[:min(pagesReadPerQuery, len(orderedDocuments))],
	)
}
