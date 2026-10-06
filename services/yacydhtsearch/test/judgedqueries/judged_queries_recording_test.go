package judgedqueries_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/peerdirectory"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryfindings"
	"github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/queryreading"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const (
	recordingSwitch = "YACYDHTSEARCH_RECORD_JUDGED_QUERIES"

	pagesReadPerQueryCeiling = 50
	recordingPagesBudget     = 10 * time.Second
)

type recordedQuery struct {
	text         string
	languageCode string
}

var recordedQueries = []recordedQuery{
	{"wikipedia", ""},
	{"debian", "en"},
	{"kubernetes", "en"},
	{"bundestag", "de"},
	{"rust", "en"},
	{"photosynthesis", "en"},
	{"openstreetmap", ""},
	{"heise", "de"},
	{"fahrrad", "de"},
	{"mastodon", ""},
	{"mercury", "en"},
	{"python", "en"},
	{"jaguar", "en"},
	{"apple", "en"},
	{"golang concurrency", "en"},
	{"python asyncio", "en"},
	{"linux kernel", "en"},
	{"arch wiki", "en"},
	{"berlin wetter", "de"},
	{"climate change", "en"},
	{"docker compose", "en"},
	{"nextcloud installation", "en"},
	{"raspberry pi", "en"},
	{"git rebase", "en"},
	{"how to install debian", "en"},
	{"how do i reset my router", "en"},
	{"why does my laptop battery drain fast", "en"},
	{"how to fix a slow computer", "en"},
	{"what is the best way to learn a new language", "en"},
	{"how to remove a stripped screw", "en"},
	{"why do cats knead blankets", "en"},
	{"how to negotiate a salary raise", "en"},
	{"what to do when locked out of your house", "en"},
	{"free software foundation", "en"},
	{"open source search engine", "en"},
	{"deutsche bahn fahrplan", "de"},
	{"static site generator", "en"},
	{"tor browser download", "en"},
	{"wolfgang amadeus mozart", "en"},
	{"machine learning tutorial", "en"},
	{"yacy peer to peer search", "en"},
	{"self hosted email server", "en"},
	{"gnu emacs manual", "en"},
	{"chaos computer club", "de"},
	{"apache httpd", "en"},
	{"postgresql documentation", "en"},
	{"libreoffice", "en"},
	{"fahrradwerkstatt berlin", "de"},
	{"how does public key cryptography work", "en"},
	{"difference between tcp and udp", "en"},
	{"best practices for database indexing", "en"},
	{"how does dns resolution work", "en"},
	{"why is my wifi connection slow", "en"},
	{"what causes climate change", "en"},
	{"how do vaccines work", "en"},
	{"advantages of renewable energy sources", "en"},
	{"how does a search engine rank pages", "en"},
	{"what is the difference between http and https", "en"},
	{"recette crêpes", "fr"},
	{"programación funcional", "es"},
	{"programmazione funzionale", "it"},
	{"сборка ядра linux", "ru"},
	{"wie funktioniert ein elektroauto", "de"},
	{"berliner mauer geschichte", "de"},
	{"comment apprendre le français", "fr"},
	{"recette tarte tatin", "fr"},
	{"como funciona internet", "es"},
	{"mejores playas de españa", "es"},
	{"storia antica di roma", "it"},
	{"как приготовить борщ", "ru"},
	{"payday loan", "en"},
	{"viagra", "en"},
	{"sildenafil", "en"},
	{"online gambling", "en"},
	{"weight loss", "en"},
	{"minecraft download", "en"},
	{"watch movies online", "en"},
	{"how to make money online", "en"},
	{"arch linux", "en"},
	{"open street map", "en"},
	{"next cloud", "en"},
	{"libre office", "en"},
	{"free bsd", "en"},
	{"home assistant", "en"},
	{"pi hole", "en"},
	{"word press", "en"},
	{"git hub", "en"},
	{"self hosting", "en"},
	{"nginx reverse proxy", "en"},
	{"ssh key generation", "en"},
	{"python virtual environment", "en"},
	{"docker volume permissions", "en"},
	{"rust ownership", "en"},
	{"vim keybindings", "en"},
	{"tls certificate renewal", "en"},
	{"bash string manipulation", "en"},
	{"sqlite full text search", "en"},
	{"kernel module signing", "en"},
	{"how does garbage collection work", "en"},
	{"how to set up a wireguard vpn", "en"},
	{"what is a bloom filter", "en"},
	{"difference between process and thread", "en"},
	{"how to write a systemd service", "en"},
	{"how to compile the linux kernel", "en"},
	{"why is my disk full", "en"},
	{"how to recover a deleted file", "en"},
	{"wie funktioniert eine wärmepumpe", "de"},
	{"was ist ein vpn", "de"},
	{"comment installer linux", "fr"},
	{"cómo aprender python", "es"},
	{"come funziona bitcoin", "it"},
	{"как настроить ssh", "ru"},
	{"debian wiki", "en"},
	{"gentoo handbook", "en"},
	{"rust book", "en"},
	{"gnome shell extensions", "en"},
	{"cheap web hosting", "en"},
	{"best vpn service", "en"},
	{"xqzvptl kwrmbz", ""},
	{"zzyzx quanternion glomph", ""},
	{"buy tadalafil online", "en"},
	{"tadalafil side effects", "en"},
	{"cheap ozempic without prescription", "en"},
	{"how does semaglutide work", "en"},
	{"best online casino bonus", "en"},
	{"gambling addiction help", "en"},
	{"sports betting tips", "en"},
	{"how do betting odds work", "en"},
	{"slot gacor", "id"},
	{"problem gambling statistics", "en"},
	{"crypto trading signals", "en"},
	{"how does a blockchain work", "en"},
	{"fast loan bad credit", "en"},
	{"how to improve credit score", "en"},
	{"detox cleanse weight loss", "en"},
	{"calorie deficit explained", "en"},
	{"replica rolex", "en"},
	{"how to spot a fake watch", "en"},
	{"essay writing service", "en"},
	{"how to write an essay", "en"},
	{"nhs online pharmacy", "en"},
	{"credit union personal loan", "en"},
	{"bitcoin exchange", "en"},
	{"gamstop", "en"},
	{"gamcare", "en"},
	{"gambling commission", "en"},
	{"coinbase", "en"},
	{"binance", "en"},
	{"experian credit report", "en"},
	{"best unsecured loans", "en"},
	{"erectile dysfunction", "en"},
	{"prescription prices", "en"},
	{"glp-1 medicines", "en"},
	{"essay writing guide", "en"},
	{"dissertation help", "en"},
	{"homework help", "en"},
	{"kredit ohne schufa", "de"},
	{"instant loan", "en"},
	{"buy xanax", "en"},
	{"tramadol", "en"},
	{"modafinil", "en"},
	{"emergency locksmith", "en"},
	{"phentermine", "en"},
	{"kamagra", "en"},
	{"keto pills", "en"},
	{"cbd oil", "en"},
	{"testosterone booster", "en"},
	{"obat kuat", "id"},
	{"потенция", "ru"},
	{"buy instagram followers", "en"},
	{"photoshop crack", "en"},
	{"windows activator", "en"},
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
		t.Run(query.text, func(t *testing.T) {
			recording.recordOne(t, query.text, query.language(t))
		})
	}
}

type judgedQueryRecording struct {
	spread     querySpread
	fetching   pageFetching
	extraction pageExtraction
	directory  *peerdirectory.Directory
}

func (query recordedQuery) language(t *testing.T) yacymodel.Language {
	t.Helper()

	if query.languageCode == "" {
		return yacymodel.Language{}
	}
	language, err := yacymodel.ParseLanguage(query.languageCode)
	if err != nil {
		t.Fatalf("the language of %q: %v", query.text, err)
	}

	return language
}

func (recording judgedQueryRecording) recordOne(
	t *testing.T, query string, language yacymodel.Language,
) {
	t.Helper()

	findings := recording.findingsOf(t, query, language)
	pages := recording.pagesReadFor(t, findings)
	writeStoredPagesOf(t, query, pages)
	findingsAndPageContents := findingsAndPageContentsOf(
		findings,
		recording.extraction.pageContentsPerDocumentOf(
			t.Context(),
			queryreading.QueryFrom(query, language).Words,
			findings,
			pagePerAddressOf(pages),
		),
	)
	writeRecordedFindingsFile(
		t,
		recordedFindingsFileOf(query),
		recordedFindingsOf(query, language, findingsAndPageContents),
	)
	judgments := writeJudgmentsOf(t, query, findingsAndPageContents)
	t.Logf("%q read the page of %d documents, judges %d, and waits for %d grades",
		query,
		len(findingsAndPageContents.pageContentsPerDocument),
		len(judgments.JudgedDocuments),
		judgments.amountOfUngradedDocuments(),
	)
}

func (recording judgedQueryRecording) findingsOf(
	t *testing.T, query string, language yacymodel.Language,
) queryfindings.Findings {
	t.Helper()

	ctx, stopQueryBudget := context.WithTimeout(t.Context(), queryBudget)
	defer stopQueryBudget()

	return recording.spread.SpreadOverPeers(
		ctx,
		queryreading.QueryFrom(query, language),
		recording.directory.AskablePeers(ctx),
	)
}

func (recording judgedQueryRecording) pagesReadFor(
	t *testing.T, findings queryfindings.Findings,
) []storedPage {
	t.Helper()

	orderedDocuments := defaultRelevanceOrdering().OrderedDocumentsOf(findings)

	return recording.fetching.fetchedPagesOf(
		t.Context(), orderedDocuments[:min(pagesReadPerQueryCeiling, len(orderedDocuments))],
	)
}
