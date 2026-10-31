package urlpostingpurge_test

import (
	"context"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel/urlhashtest"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/pageversion"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlmeta"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlpostingpurge"
)

type recordedPurges struct {
	postingsPerURL map[yacymodel.URLHash]int
	urls           []yacymodel.URLHash
}

func (r *recordedPurges) ObservePostingsPurgedWithURL(
	_ *vault.Txn,
	url yacymodel.URLHash,
	amountOfPostings int,
) {
	r.urls = append(r.urls, url)
	r.postingsPerURL[url] += amountOfPostings
}

type harness struct {
	vault     *vault.Vault
	index     rwipostings.PostingIndex
	writer    rwipostings.PostingWriter
	urls      urlmeta.URLMetadataWriter
	evictor   urlmeta.URLEvictor
	observing *recordedPurges
}

func openHarness(t *testing.T) harness {
	t.Helper()

	v, err := memoryvault.Open(0, nil, wallclock.Clock{})
	if err != nil {
		t.Fatalf("memoryvault.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	index, writer, purger, err := rwipostings.Open(
		context.Background(),
		v,
		yacymodel.DHTRingPartitions{},
		nil,
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}
	observing := &recordedPurges{postingsPerURL: map[yacymodel.URLHash]int{}}
	_, evictor, urlWriter, err := urlmeta.Open(
		context.Background(),
		v,
		nil,
		urlpostingpurge.New(purger, observing),
	)
	if err != nil {
		t.Fatalf("urlmeta.Open: %v", err)
	}

	return harness{
		vault:     v,
		index:     index,
		writer:    writer,
		urls:      urlWriter,
		evictor:   evictor,
		observing: observing,
	}
}

func urlHash(t *testing.T, seed string) yacymodel.URLHash {
	t.Helper()

	return urlhashtest.URLHashOf(t, "http://example.com/"+seed)
}

func (h harness) storeMetadata(t *testing.T, seed string) yacymodel.URLHash {
	t.Helper()

	address := "http://example.com/" + seed
	hash := urlHash(t, seed)
	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return h.urls.Write(
			tx,
			yacymodel.URLMetadata{Hash: hash, Address: address},
			pageversion.SourceThisNode(),
		)
	}); err != nil {
		t.Fatalf("urls.Write: %v", err)
	}

	return hash
}

func (h harness) writePosting(t *testing.T, word string, url yacymodel.URLHash) {
	t.Helper()

	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return h.writer.Write(tx, yacymodel.RWIPosting{
			WordHash: yacymodel.WordHash(word),
			URLHash:  url,
			Language: yacymodel.LanguageOfUndeclaredDocument,
			Hits:     1,
		})
	}); err != nil {
		t.Fatalf("Write: %v", err)
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

func TestPurgedURLTakesItsPostingsAndReportsHowMany(t *testing.T) {
	h := openHarness(t)
	purged := h.storeMetadata(t, "u1")
	words := []string{"w1", "w2", "w3"}
	for _, word := range words {
		h.writePosting(t, word, purged)
	}

	h.purgeMetadata(t, purged)

	for _, word := range words {
		if h.indexed(t, word, purged) {
			t.Errorf("posting of %q outlived the url metadata it describes", word)
		}
	}
	if got := h.observing.postingsPerURL[purged]; got != len(words) {
		t.Errorf("postings purged = %d, want the %d the url had", got, len(words))
	}
}
