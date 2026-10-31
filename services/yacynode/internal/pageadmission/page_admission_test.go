package pageadmission_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/documentextraction"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapecontract"
	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/urlhashtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagerwi"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlpostingpurge"
)

const (
	busyPause   = 5 * time.Second
	pageAddress = "http://example.com/page"
)

var errWriteRefused = errors.New("write refused")

func ringPartitions(t *testing.T) yacymodel.DHTRingPartitions {
	t.Helper()
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatal(err)
	}

	return partitions
}

type discardedPurges struct{}

func (discardedPurges) ObservePostingsPurgedWithURL(*vault.Txn, yacymodel.URLHash, int) {}

type refusedWriter struct {
	writer rwipostings.PostingWriter
	word   yacymodel.Hash
}

func (r refusedWriter) Write(tx *vault.Txn, pagePostings yacymodel.PagePostings) error {
	for _, posting := range pagePostings.Postings() {
		if posting.WordHash == r.word {
			return errWriteRefused
		}
	}

	return r.writer.Write(tx, pagePostings)
}

type noStopwords struct{}

func (noStopwords) Contains(yacymodel.Hash, yacymodel.Language) bool {
	return false
}

type harness struct {
	vault     *vault.Vault
	index     rwipostings.PostingIndex
	directory urlmeta.URLDirectory
	receiver  pageadmission.PageReceiver
}

func openHarness(t *testing.T, quotaBytes int64) harness {
	t.Helper()

	return openHarnessWithWriter(t, quotaBytes, nil)
}

func openHarnessWithWriter(
	t *testing.T,
	quotaBytes int64,
	wrapWriter func(rwipostings.PostingWriter) rwipostings.PostingWriter,
) harness {
	t.Helper()

	v, err := memoryvault.Open(quotaBytes, nil, wallclock.Clock{})
	if err != nil {
		t.Fatalf("memoryvault.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	index, writer, purger, _, err := rwipostings.Open(
		context.Background(),
		v,
		ringPartitions(t),
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}
	if wrapWriter != nil {
		writer = wrapWriter(writer)
	}
	directory, evictor, metadataWriter, err := urlmeta.Open(
		context.Background(),
		v,
		nil,
		urlpostingpurge.New(purger, discardedPurges{}),
	)
	if err != nil {
		t.Fatalf("urlmeta.Open: %v", err)
	}

	return harness{
		vault:     v,
		index:     index,
		directory: directory,
		receiver: pageadmission.Open(
			v,
			evictor,
			metadataWriter,
			writer,
			pageadmission.Config{Pause: busyPause},
		),
	}
}

type pageWord struct {
	word string
	hits uint8
}

func pageTitled(
	t *testing.T,
	title string,
	words ...pageWord,
) yacymodel.Optional[pagerwi.PageRWI] {
	t.Helper()

	textWords := []string{}
	for _, word := range words {
		for range word.hits {
			textWords = append(textWords, word.word)
		}
	}
	text := strings.Join(textWords, " ")
	scrapedPage := pagescrapecontract.OfferedPage{
		PageURL:     canonicalurltest.CanonicalURLOf(t, pageAddress),
		LandedURL:   canonicalurltest.CanonicalURLOf(t, pageAddress),
		ContentType: "text/plain",
		Body:        []byte(text),
	}

	return pagerwi.NewIndexBuilder(noStopwords{}, ringPartitions(t)).RWIFrom(
		t.Context(),
		scrapedPage,
		documentextraction.Document{Title: title},
		[]byte(text),
		time.Unix(1_700_000_000, 0),
	)
}

func pageURLHash(t *testing.T) yacymodel.URLHash {
	t.Helper()

	return urlhashtest.URLHashOf(t, pageAddress)
}

func (h harness) receive(
	t *testing.T,
	pageRWI yacymodel.Optional[pagerwi.PageRWI],
) pageadmission.Receipt {
	t.Helper()

	receipt, err := h.receiver.Receive(context.Background(), pageURLHash(t), pageRWI)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}

	return receipt
}

func (h harness) storedPosting(t *testing.T, word string) (yacymodel.RWIPosting, bool) {
	t.Helper()

	var (
		stored yacymodel.RWIPosting
		found  bool
	)
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		posting, indexed, err := h.index.PostingOf(
			tx,
			yacymodel.WordHash(word),
			pageURLHash(t),
		)
		stored, found = posting, indexed

		return err
	}); err != nil {
		t.Fatalf("PostingOf: %v", err)
	}

	return stored, found
}

func (h harness) storedTitle(t *testing.T) string {
	t.Helper()
	metadata, _ := h.storedMetadata(t)

	return metadata.Title
}

func (h harness) storedMetadata(t *testing.T) (yacymodel.URLMetadata, bool) {
	t.Helper()

	var rows map[yacymodel.URLHash]yacymodel.URLMetadata
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		stored, err := h.directory.MetadataPerHash(
			tx,
			[]yacymodel.URLHash{pageURLHash(t)},
		)
		rows = stored

		return err
	}); err != nil {
		t.Fatalf("MetadataPerHash: %v", err)
	}
	metadata, stored := rows[pageURLHash(t)]

	return metadata, stored
}

func TestRecrawledPageReplacesEveryPostingItHeld(t *testing.T) {
	h := openHarness(t, 0)
	crawled := pageTitled(t, "before", pageWord{"w1", 1}, pageWord{"w2", 3})
	h.receive(t, crawled)

	recrawled := pageTitled(t, "after", pageWord{"w2", 7}, pageWord{"w3", 2})
	h.receive(t, recrawled)

	if _, found := h.storedPosting(t, "w1"); found {
		t.Error("a word the recrawled page no longer has kept its posting")
	}
	if _, found := h.storedPosting(t, "w3"); !found {
		t.Error("a word the recrawled page gained has no posting")
	}
	kept, found := h.storedPosting(t, "w2")
	if !found || kept.Hits != 7 {
		t.Errorf("posting of the kept word = %+v (found %t), want the recrawled 7 hits",
			kept, found)
	}
	if got := h.storedTitle(t); got != "after" {
		t.Errorf("stored title = %q, want the recrawled title", got)
	}
}

func TestRecrawledPageWithoutWordsLeavesNothingOfThePageBehind(t *testing.T) {
	h := openHarness(t, 0)
	h.receive(t, pageTitled(t, "before", pageWord{"w1", 1}))

	h.receive(t, pageTitled(t, "after"))

	if _, found := h.storedPosting(t, "w1"); found {
		t.Error("a page with no words left the posting of a word it had before")
	}
	if metadata, stored := h.storedMetadata(t); stored {
		t.Errorf("stored metadata = %+v, want none for a page without words", metadata)
	}
}

func TestRefusedPostingLeavesThePageTheNodeAlreadyHeld(t *testing.T) {
	h := openHarnessWithWriter(t, 0,
		func(writer rwipostings.PostingWriter) rwipostings.PostingWriter {
			return refusedWriter{writer: writer, word: yacymodel.WordHash("w3")}
		})
	crawled := pageTitled(t, "before", pageWord{"w1", 1}, pageWord{"w2", 3})
	h.receive(t, crawled)

	recrawled := pageTitled(t, "after", pageWord{"w2", 7}, pageWord{"w3", 1})
	if _, err := h.receiver.Receive(
		context.Background(),
		pageURLHash(t),
		recrawled,
	); !errors.Is(err, errWriteRefused) {
		t.Fatalf("Receive error = %v, want the refused write", err)
	}

	if _, found := h.storedPosting(t, "w1"); !found {
		t.Error("a refused recrawl purged the posting of a word the page had before")
	}
	kept, found := h.storedPosting(t, "w2")
	if !found || kept.Hits != 3 {
		t.Errorf("posting of the kept word = %+v (found %t), want the 3 hits stored before",
			kept, found)
	}
	if got := h.storedTitle(t); got != "before" {
		t.Errorf("stored title = %q, want the title stored before the refusal", got)
	}
}

func TestReceiveBusyAtCapacity(t *testing.T) {
	h := openHarness(t, usedBytesOfAnEmptyNode(t)+1)
	crawled := pageTitled(t, "before", pageWord{"w1", 1})
	h.receive(t, crawled)

	recrawled := pageTitled(t, "after", pageWord{"w2", 1})
	receipt := h.receive(t, recrawled)

	if !receipt.Busy || receipt.Pause != busyPause {
		t.Fatalf("receipt = %+v, want Busy with the configured pause", receipt)
	}
	if _, found := h.storedPosting(t, "w1"); !found {
		t.Error("a refused page purged the postings of the page the node held")
	}
	if _, found := h.storedPosting(t, "w2"); found {
		t.Error("a refused page left a posting behind")
	}
	if got := h.storedTitle(t); got != "before" {
		t.Errorf("stored title = %q, want the title stored before the refusal", got)
	}
}

func usedBytesOfAnEmptyNode(t *testing.T) int64 {
	t.Helper()

	usedBytes, err := openHarness(t, 0).vault.UsedBytes(context.Background())
	if err != nil {
		t.Fatalf("UsedBytes: %v", err)
	}

	return usedBytes
}
