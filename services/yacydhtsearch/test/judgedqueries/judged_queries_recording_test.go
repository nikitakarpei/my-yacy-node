package judgedqueries_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryanswers"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/searchquery"
)

const (
	recordingSwitch = "YACYDHTSEARCH_RECORD_JUDGED_QUERIES"

	pagesReadPerQuery    = 50
	recordingPagesBudget = 10 * time.Second
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
	"arch linux",
	"open street map",
	"next cloud",
	"libre office",
	"free bsd",
	"home assistant",
	"pi hole",
	"word press",
	"git hub",
	"self hosting",
	"nginx reverse proxy",
	"ssh key generation",
	"python virtual environment",
	"docker volume permissions",
	"rust ownership",
	"vim keybindings",
	"tls certificate renewal",
	"bash string manipulation",
	"sqlite full text search",
	"kernel module signing",
	"how does garbage collection work",
	"how to set up a wireguard vpn",
	"what is a bloom filter",
	"difference between process and thread",
	"how to write a systemd service",
	"how to compile the linux kernel",
	"why is my disk full",
	"how to recover a deleted file",
	"wie funktioniert eine wärmepumpe",
	"was ist ein vpn",
	"comment installer linux",
	"cómo aprender python",
	"come funziona bitcoin",
	"как настроить ssh",
	"debian wiki",
	"gentoo handbook",
	"rust book",
	"gnome shell extensions",
	"cheap web hosting",
	"best vpn service",
	"xqzvptl kwrmbz",
	"zzyzx quanternion glomph",
	"buy tadalafil online",
	"tadalafil side effects",
	"cheap ozempic without prescription",
	"how does semaglutide work",
	"best online casino bonus",
	"gambling addiction help",
	"sports betting tips",
	"how do betting odds work",
	"slot gacor",
	"problem gambling statistics",
	"crypto trading signals",
	"how does a blockchain work",
	"fast loan bad credit",
	"how to improve credit score",
	"detox cleanse weight loss",
	"calorie deficit explained",
	"replica rolex",
	"how to spot a fake watch",
	"essay writing service",
	"how to write an essay",
	"nhs online pharmacy",
	"credit union personal loan",
	"bitcoin exchange",
	"gamstop",
	"gamcare",
	"gambling commission",
	"coinbase",
	"binance",
	"experian credit report",
	"best unsecured loans",
	"erectile dysfunction",
	"prescription prices",
	"glp-1 medicines",
	"essay writing guide",
	"dissertation help",
	"homework help",
	"kredit ohne schufa",
	"instant loan",
	"buy xanax",
	"tramadol",
	"modafinil",
	"emergency locksmith",
	"phentermine",
	"kamagra",
	"keto pills",
	"cbd oil",
	"testosterone booster",
	"obat kuat",
	"потенция",
	"buy instagram followers",
	"photoshop crack",
	"windows activator",
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
	pages := recording.pagesReadFor(t, answers)
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

func (recording judgedQueryRecording) pagesReadFor(
	t *testing.T, answers queryanswers.AnsweredQuery,
) []storedPage {
	t.Helper()

	orderedDocuments := defaultRelevanceOrdering().OrderedDocumentsOf(answers)

	return recording.fetching.fetchedPagesOf(
		t.Context(), orderedDocuments[:min(pagesReadPerQuery, len(orderedDocuments))],
	)
}
