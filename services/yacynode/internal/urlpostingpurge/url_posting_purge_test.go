package urlpostingpurge_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlpostingpurge"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlreferences"
)

type recordedPurges struct {
	postingsPerURL map[yacymodel.URLHash]int
	urls           []yacymodel.URLHash
}

func (r *recordedPurges) ObservePostingsPurgedWithURL(
	url yacymodel.URLHash,
	amountOfPostings int,
) {
	r.urls = append(r.urls, url)
	r.postingsPerURL[url] += amountOfPostings
}

type harness struct {
	vault     *vault.Vault
	index     rwipostings.PostingIndex
	admitter  rwipostings.PostingAdmitter
	urls      urlmeta.URLReceiver
	evictor   urlmeta.URLEvictor
	observing *recordedPurges
}

func openHarness(t *testing.T) harness {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
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
	observing := &recordedPurges{postingsPerURL: map[yacymodel.URLHash]int{}}
	_, evictor, receiver, err := urlmeta.Open(
		v,
		urlpostingpurge.New(references, purger, observing),
	)
	if err != nil {
		t.Fatalf("urlmeta.Open: %v", err)
	}

	return harness{
		vault:     v,
		index:     index,
		admitter:  admitter,
		urls:      receiver,
		evictor:   evictor,
		observing: observing,
	}
}

func urlHash(t *testing.T, seed string) yacymodel.URLHash {
	t.Helper()

	address, err := url.Parse("http://example.com/" + seed)
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}

	return yacymodel.URLNormalformOf(address).Hash()
}

func (h harness) storeMetadata(t *testing.T, seed string) yacymodel.URLHash {
	t.Helper()

	address := "http://example.com/" + seed
	hash := urlHash(t, seed)
	if _, err := h.urls.Receive(context.Background(), []yacymodel.URLMetadata{
		{Hash: hash, Address: address},
	}); err != nil {
		t.Fatalf("urls.Receive: %v", err)
	}

	return hash
}

func (h harness) admitPosting(t *testing.T, word string, url yacymodel.URLHash) {
	t.Helper()

	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return h.admitter.Admit(tx, yacymodel.RWIPosting{
			WordHash: yacymodel.WordHash(word),
			URLHash:  url,
			Language: yacymodel.LanguageOfUndeclaredDocument,
			Hits:     1,
		})
	}); err != nil {
		t.Fatalf("Admit: %v", err)
	}
}

func (h harness) purgeMetadata(t *testing.T, url yacymodel.URLHash) {
	t.Helper()

	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		_, err := h.evictor.Purge(context.Background(), tx, []yacymodel.URLHash{url})

		return err
	}); err != nil {
		t.Fatalf("Purge: %v", err)
	}
}

func (h harness) indexed(t *testing.T, word string, url yacymodel.URLHash) bool {
	t.Helper()

	var found bool
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		_, stored, err := h.index.PostingOf(tx, yacymodel.WordHash(word), url)
		found = stored

		return err
	}); err != nil {
		t.Fatalf("PostingOf: %v", err)
	}

	return found
}

func TestPurgedURLKeepsNoPosting(t *testing.T) {
	h := openHarness(t)
	purged := h.storeMetadata(t, "u1")
	h.admitPosting(t, "w1", purged)
	h.admitPosting(t, "w2", purged)

	h.purgeMetadata(t, purged)

	for _, word := range []string{"w1", "w2"} {
		if h.indexed(t, word, purged) {
			t.Errorf("posting of %q outlived the url metadata it describes", word)
		}
	}
}

func TestPurgedURLLeavesThePostingsOfEveryOtherURL(t *testing.T) {
	h := openHarness(t)
	purged, kept := h.storeMetadata(t, "u1"), h.storeMetadata(t, "u2")
	h.admitPosting(t, "w1", purged)
	h.admitPosting(t, "w1", kept)

	h.purgeMetadata(t, purged)

	if !h.indexed(t, "w1", kept) {
		t.Error("purging one url dropped the posting of another")
	}
}

func TestPurgedURLWithoutPostingsIsPurgedCleanly(t *testing.T) {
	h := openHarness(t)
	purged := h.storeMetadata(t, "u1")

	h.purgeMetadata(t, purged)

	if got := h.observing.postingsPerURL[purged]; got != 0 {
		t.Errorf("postings purged = %d, want 0", got)
	}
	if len(h.observing.urls) != 1 || h.observing.urls[0] != purged {
		t.Errorf("observed urls = %v, want the purged url", h.observing.urls)
	}
}

func TestObserverCountsThePostingsPurgedWithTheURL(t *testing.T) {
	h := openHarness(t)
	purged := h.storeMetadata(t, "u1")
	h.admitPosting(t, "w1", purged)
	h.admitPosting(t, "w2", purged)
	h.admitPosting(t, "w3", purged)

	h.purgeMetadata(t, purged)

	if got := h.observing.postingsPerURL[purged]; got != 3 {
		t.Errorf("postings purged = %d, want the 3 the url had", got)
	}
}
