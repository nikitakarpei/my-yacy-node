package pageadmission_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pagerwi"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlpostingpurge"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlreferences"
)

const (
	busyPause   = 5 * time.Second
	pageAddress = "http://example.com/page"
)

var errAdmissionRefused = errors.New("admission refused")

type discardedPurges struct{}

func (discardedPurges) ObservePostingsPurgedWithURL(yacymodel.URLHash, int) {}

type refusedAdmitter struct {
	admitter rwipostings.PostingAdmitter
	word     yacymodel.Hash
}

func (r refusedAdmitter) Admit(tx *vault.Txn, posting yacymodel.RWIPosting) error {
	if posting.WordHash == r.word {
		return errAdmissionRefused
	}

	return r.admitter.Admit(tx, posting)
}

type harness struct {
	vault     *vault.Vault
	index     rwipostings.PostingIndex
	directory urlmeta.URLDirectory
	receiver  pageadmission.PageReceiver
}

func openHarness(t *testing.T, quotaBytes int64) harness {
	t.Helper()

	return openHarnessWithAdmitter(t, quotaBytes, nil)
}

func openHarnessWithAdmitter(
	t *testing.T,
	quotaBytes int64,
	wrapAdmitter func(rwipostings.PostingAdmitter) rwipostings.PostingAdmitter,
) harness {
	t.Helper()

	v, err := memoryvault.Open(quotaBytes, nil)
	if err != nil {
		t.Fatalf("memoryvault.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	references, err := urlreferences.Open(v)
	if err != nil {
		t.Fatalf("urlreferences.Open: %v", err)
	}
	index, admitter, purger, err := rwipostings.Open(v, references)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}
	if wrapAdmitter != nil {
		admitter = wrapAdmitter(admitter)
	}
	directory, evictor, metadataAdmitter, _, err := urlmeta.Open(
		v,
		urlpostingpurge.New(references, purger, discardedPurges{}),
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
			metadataAdmitter,
			admitter,
			pageadmission.Config{Pause: busyPause},
		),
	}
}

type pageWord struct {
	word string
	hits int
}

func pageTitled(t *testing.T, title string, words ...pageWord) pagerwi.PageRWI {
	t.Helper()

	hash, err := yacymodel.URLHashOf(pageAddress)
	if err != nil {
		t.Fatalf("URLHashOf: %v", err)
	}

	postings := make([]yacymodel.RWIPosting, 0, len(words))
	for _, word := range words {
		postings = append(postings, yacymodel.RWIPosting{
			WordHash: yacymodel.WordHash(word.word),
			URLHash:  hash,
			Language: yacymodel.LanguageOfUndeclaredDocument,
			Hits:     word.hits,
		})
	}

	return pagerwi.PageRWI{
		Metadata: yacymodel.URLMetadata{Hash: hash, Address: pageAddress, Title: title},
		Postings: postings,
	}
}

func (h harness) receive(t *testing.T, page pagerwi.PageRWI) pageadmission.Receipt {
	t.Helper()

	receipt, err := h.receiver.Receive(context.Background(), page)
	if err != nil {
		t.Fatalf("Receive: %v", err)
	}

	return receipt
}

func (h harness) storedPosting(
	t *testing.T,
	word string,
	page pagerwi.PageRWI,
) (yacymodel.RWIPosting, bool) {
	t.Helper()

	var (
		stored yacymodel.RWIPosting
		found  bool
	)
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		posting, indexed, err := h.index.PostingOf(
			tx,
			yacymodel.WordHash(word),
			page.Metadata.Hash,
		)
		stored, found = posting, indexed

		return err
	}); err != nil {
		t.Fatalf("PostingOf: %v", err)
	}

	return stored, found
}

func (h harness) storedTitle(t *testing.T, page pagerwi.PageRWI) string {
	t.Helper()

	var rows map[yacymodel.URLHash]yacymodel.URLMetadata
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		stored, err := h.directory.MetadataPerHash(
			tx,
			[]yacymodel.URLHash{page.Metadata.Hash},
		)
		rows = stored

		return err
	}); err != nil {
		t.Fatalf("MetadataPerHash: %v", err)
	}

	return rows[page.Metadata.Hash].Title
}

func TestRecrawledPageReplacesEveryPostingItHeld(t *testing.T) {
	h := openHarness(t, 0)
	crawled := pageTitled(t, "before", pageWord{"w1", 1}, pageWord{"w2", 3})
	h.receive(t, crawled)

	recrawled := pageTitled(t, "after", pageWord{"w2", 7}, pageWord{"w3", 2})
	h.receive(t, recrawled)

	if _, found := h.storedPosting(t, "w1", recrawled); found {
		t.Error("a word the recrawled page no longer has kept its posting")
	}
	if _, found := h.storedPosting(t, "w3", recrawled); !found {
		t.Error("a word the recrawled page gained has no posting")
	}
	kept, found := h.storedPosting(t, "w2", recrawled)
	if !found || kept.Hits != 7 {
		t.Errorf("posting of the kept word = %+v (found %t), want the recrawled 7 hits",
			kept, found)
	}
	if got := h.storedTitle(t, recrawled); got != "after" {
		t.Errorf("stored title = %q, want the recrawled title", got)
	}
}

func TestRecrawledPageWithoutWordsLeavesNoPostingBehind(t *testing.T) {
	h := openHarness(t, 0)
	crawled := pageTitled(t, "before", pageWord{"w1", 1})
	h.receive(t, crawled)

	wordless := pageTitled(t, "after")
	h.receive(t, wordless)

	if _, found := h.storedPosting(t, "w1", wordless); found {
		t.Error("a page with no words left the posting of a word it had before")
	}
	if got := h.storedTitle(t, wordless); got != "after" {
		t.Errorf("stored title = %q, want the recrawled title", got)
	}
}

func TestRefusedPostingLeavesThePageTheNodeAlreadyHeld(t *testing.T) {
	h := openHarnessWithAdmitter(t, 0,
		func(admitter rwipostings.PostingAdmitter) rwipostings.PostingAdmitter {
			return refusedAdmitter{admitter: admitter, word: yacymodel.WordHash("w3")}
		})
	crawled := pageTitled(t, "before", pageWord{"w1", 1}, pageWord{"w2", 3})
	h.receive(t, crawled)

	recrawled := pageTitled(t, "after", pageWord{"w2", 7}, pageWord{"w3", 1})
	if _, err := h.receiver.Receive(
		context.Background(),
		recrawled,
	); !errors.Is(err, errAdmissionRefused) {
		t.Fatalf("Receive error = %v, want the refused admission", err)
	}

	if _, found := h.storedPosting(t, "w1", crawled); !found {
		t.Error("a refused recrawl purged the posting of a word the page had before")
	}
	kept, found := h.storedPosting(t, "w2", crawled)
	if !found || kept.Hits != 3 {
		t.Errorf("posting of the kept word = %+v (found %t), want the 3 hits stored before",
			kept, found)
	}
	if got := h.storedTitle(t, crawled); got != "before" {
		t.Errorf("stored title = %q, want the title stored before the refusal", got)
	}
}

func TestReceiveBusyAtCapacity(t *testing.T) {
	h := openHarness(t, 1)
	crawled := pageTitled(t, "before", pageWord{"w1", 1})
	h.receive(t, crawled)

	recrawled := pageTitled(t, "after", pageWord{"w2", 1})
	receipt := h.receive(t, recrawled)

	if !receipt.Busy || receipt.Pause != busyPause {
		t.Fatalf("receipt = %+v, want Busy with the configured pause", receipt)
	}
	if _, found := h.storedPosting(t, "w1", crawled); !found {
		t.Error("a refused page purged the postings of the page the node held")
	}
	if _, found := h.storedPosting(t, "w2", recrawled); found {
		t.Error("a refused page left a posting behind")
	}
	if got := h.storedTitle(t, crawled); got != "before" {
		t.Errorf("stored title = %q, want the title stored before the refusal", got)
	}
}
