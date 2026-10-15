package urlpageversion_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vault/vaultenginetest"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwipostings"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/urlpageversion"
)

type countingVerdicts map[urlpageversion.Verdict]int

func (c countingVerdicts) ObserveVerdict(tx *vault.Txn, verdict urlpageversion.Verdict) {
	tx.RunAfterCommit(func() { c[verdict]++ })
}

type harness struct {
	vault    *vault.Vault
	index    rwipostings.PostingIndex
	versions *urlpageversion.PageVersions
	verdicts countingVerdicts
	clock    time.Time
}

var (
	firstOrigin  = rwiorigin.Of(yacymodel.WordHash("first peer"), "203.0.113.7")
	secondOrigin = rwiorigin.Of(yacymodel.WordHash("second peer"), "198.51.100.9")
)

func openHarness(t *testing.T) *harness {
	t.Helper()

	v, err := vault.New(vaultenginetest.EngineRepeatingWrites(memoryvault.OpenEngine(0)), nil)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	index, references, admitter, purger, err := rwipostings.Open(
		context.Background(),
		v,
		yacymodel.DHTRingPartitions(1),
		nil,
	)
	if err != nil {
		t.Fatalf("rwipostings.Open: %v", err)
	}
	h := &harness{
		vault:    v,
		index:    index,
		verdicts: countingVerdicts{},
		clock:    time.Date(2026, time.October, 16, 12, 0, 0, 0, time.UTC),
	}
	versions, err := urlpageversion.Open(
		v,
		urlpageversion.Postings{
			Index:      index,
			References: references,
			Admitter:   admitter,
			Purger:     purger,
		},
		func() time.Time { return h.clock },
		urlpageversion.Config{
			SwapFloorPerHour: 1,
			SwapSharePerHour: 0,
			Observer:         h.verdicts,
		},
	)
	if err != nil {
		t.Fatalf("urlpageversion.Open: %v", err)
	}
	h.versions = versions

	return h
}

func (h *harness) today() yacymodel.MicroDate {
	return yacymodel.MicroDateFromTime(h.clock)
}

func (h *harness) admit(
	t *testing.T,
	origin rwiorigin.Origin,
	posting yacymodel.RWIPosting,
	pageVersion yacymodel.MicroDate,
) urlpageversion.Verdict {
	t.Helper()

	var verdict urlpageversion.Verdict
	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		admitted, err := h.versions.Admit(tx, origin, posting, pageVersion)
		verdict = admitted

		return err
	}); err != nil {
		t.Fatalf("Admit: %v", err)
	}

	return verdict
}

func (h *harness) purgeURL(t *testing.T, hash yacymodel.URLHash) {
	t.Helper()

	if err := h.vault.Update(context.Background(), func(tx *vault.Txn) error {
		return h.versions.URLPurged(tx, hash)
	}); err != nil {
		t.Fatalf("URLPurged: %v", err)
	}
}

func (h *harness) indexedPosting(
	t *testing.T,
	posting yacymodel.RWIPosting,
) (yacymodel.RWIPosting, bool) {
	t.Helper()

	var indexed yacymodel.RWIPosting
	var found bool
	if err := h.vault.View(context.Background(), func(tx *vault.Txn) error {
		stored, ok, err := h.index.PostingOf(tx, posting.WordHash, posting.URLHash)
		indexed, found = stored, ok

		return err
	}); err != nil {
		t.Fatalf("PostingOf: %v", err)
	}

	return indexed, found
}

func urlHash(seed string) yacymodel.URLHash {
	address, err := url.Parse("http://example.com/" + seed)
	if err != nil {
		panic(err)
	}

	return yacymodel.URLNormalformOf(address).Hash()
}

func postingOf(word string, version yacymodel.MicroDate) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{
		WordHash:     yacymodel.WordHash(word),
		URLHash:      urlHash("page"),
		LastModified: version,
		Language:     yacymodel.LanguageOfUndeclaredDocument,
		Hits:         1,
	}
}

func TestFirstPostingsOfAURLSetItsVersion(t *testing.T) {
	h := openHarness(t)
	version := h.today() - 3

	verdict := h.admit(t, firstOrigin, postingOf("alpha", version), version)

	if verdict != urlpageversion.VerdictNewURL {
		t.Fatalf("verdict = %q, want %q", verdict, urlpageversion.VerdictNewURL)
	}
	if _, found := h.indexedPosting(t, postingOf("alpha", version)); !found {
		t.Fatal("first posting of the url was not admitted")
	}
	if h.verdicts[urlpageversion.VerdictNewURL] != 1 {
		t.Fatalf("observed verdicts = %v, want one first version", h.verdicts)
	}
}

func TestOlderPostingIsRejected(t *testing.T) {
	h := openHarness(t)
	indexed := h.today() - 3
	h.admit(t, firstOrigin, postingOf("alpha", indexed), indexed)

	verdict := h.admit(t, secondOrigin, postingOf("beta", indexed-1), indexed)

	if verdict != urlpageversion.VerdictOlderVersion {
		t.Fatalf("verdict = %q, want %q", verdict, urlpageversion.VerdictOlderVersion)
	}
	if _, found := h.indexedPosting(t, postingOf("beta", indexed-1)); found {
		t.Fatal("older posting was admitted")
	}
}

func TestEqualPostingFillsInOnlyMissingWordsFromTheOrigin(t *testing.T) {
	h := openHarness(t)
	indexed := h.today() - 3
	h.admit(t, firstOrigin, postingOf("alpha", indexed), indexed)
	resent := postingOf("alpha", indexed)
	resent.Hits = 7

	for _, c := range []struct {
		name    string
		origin  rwiorigin.Origin
		posting yacymodel.RWIPosting
		want    urlpageversion.Verdict
	}{
		{name: "missing word from the origin", origin: firstOrigin, posting: postingOf("beta", indexed), want: urlpageversion.VerdictMissingWord},
		{name: "indexed word from the origin", origin: firstOrigin, posting: resent, want: urlpageversion.VerdictSameVersion},
		{name: "missing word from another origin", origin: secondOrigin, posting: postingOf("gamma", indexed), want: urlpageversion.VerdictSameVersion},
	} {
		t.Run(c.name, func(t *testing.T) {
			if verdict := h.admit(t, c.origin, c.posting, indexed); verdict != c.want {
				t.Fatalf("verdict = %q, want %q", verdict, c.want)
			}
		})
	}
	if stored, _ := h.indexedPosting(t, resent); stored.Hits != 1 {
		t.Fatalf("indexed posting hits = %d, want the first copy kept", stored.Hits)
	}
	if _, found := h.indexedPosting(t, postingOf("gamma", indexed)); found {
		t.Fatal("equal posting from another origin was admitted")
	}
}

func TestNewerPostingSwapsTheURLToItsVersion(t *testing.T) {
	h := openHarness(t)
	older, newer := h.today()-3, h.today()
	h.admit(t, firstOrigin, postingOf("alpha", older), older)
	h.admit(t, firstOrigin, postingOf("beta", older), older)

	verdict := h.admit(t, secondOrigin, postingOf("gamma", newer), newer)

	if verdict != urlpageversion.VerdictNewerVersion {
		t.Fatalf("verdict = %q, want %q", verdict, urlpageversion.VerdictNewerVersion)
	}
	for _, word := range []string{"alpha", "beta"} {
		if _, found := h.indexedPosting(t, postingOf(word, older)); found {
			t.Fatalf("posting %q of the older version survived the swap", word)
		}
	}
	if _, found := h.indexedPosting(t, postingOf("gamma", newer)); !found {
		t.Fatal("posting of the newer version was not admitted")
	}
	if verdict := h.admit(
		t,
		secondOrigin,
		postingOf("delta", newer),
		newer,
	); verdict != urlpageversion.VerdictMissingWord {
		t.Fatalf("verdict after swap = %q, want the new origin to fill in", verdict)
	}
}

func TestNewerPostingAwaitsNewerMetadataThenSwapsWhenItArrives(t *testing.T) {
	h := openHarness(t)
	older, newer := h.today()-3, h.today()
	h.admit(t, firstOrigin, postingOf("alpha", older), older)

	verdict := h.admit(t, secondOrigin, postingOf("beta", newer), older)
	if verdict != urlpageversion.VerdictNewerThanMetadata {
		t.Fatalf("verdict = %q, want %q", verdict, urlpageversion.VerdictNewerThanMetadata)
	}
	if _, found := h.indexedPosting(t, postingOf("alpha", older)); !found {
		t.Fatal("older version was purged before the newer metadata arrived")
	}

	verdict = h.admit(t, secondOrigin, postingOf("beta", newer), newer)
	if verdict != urlpageversion.VerdictNewerVersion {
		t.Fatalf("verdict = %q, want %q", verdict, urlpageversion.VerdictNewerVersion)
	}
	if _, found := h.indexedPosting(t, postingOf("alpha", older)); found {
		t.Fatal("older version survived the swap")
	}
}

func TestPostingDatedAfterTomorrowIsRejected(t *testing.T) {
	h := openHarness(t)
	future := h.today() + 2

	verdict := h.admit(t, firstOrigin, postingOf("alpha", future), future)

	if verdict != urlpageversion.VerdictFutureDate {
		t.Fatalf("verdict = %q, want %q", verdict, urlpageversion.VerdictFutureDate)
	}
	if verdict := h.admit(
		t,
		firstOrigin,
		postingOf("alpha", h.today()+1),
		h.today()+1,
	); verdict != urlpageversion.VerdictNewURL {
		t.Fatalf("verdict for tomorrow = %q, want it admitted", verdict)
	}
}

func TestSwapsPerNetworkStopAtTheHourlyLimit(t *testing.T) {
	h := openHarness(t)
	base := h.today() - 5
	h.admit(t, firstOrigin, postingOf("alpha", base), base)

	if verdict := h.admit(
		t,
		secondOrigin,
		postingOf("beta", base+1),
		base+1,
	); verdict != urlpageversion.VerdictNewerVersion {
		t.Fatalf("first swap verdict = %q, want swapped", verdict)
	}
	if verdict := h.admit(
		t,
		secondOrigin,
		postingOf("gamma", base+2),
		base+2,
	); verdict != urlpageversion.VerdictSwapLimit {
		t.Fatalf("second swap verdict = %q, want over the limit", verdict)
	}
	if verdict := h.admit(
		t,
		firstOrigin,
		postingOf("gamma", base+2),
		base+2,
	); verdict != urlpageversion.VerdictNewerVersion {
		t.Fatalf("swap from another network = %q, want swapped", verdict)
	}

	h.clock = h.clock.Add(time.Hour)

	if verdict := h.admit(
		t,
		secondOrigin,
		postingOf("delta", base+3),
		base+3,
	); verdict != urlpageversion.VerdictNewerVersion {
		t.Fatalf("swap an hour later = %q, want swapped", verdict)
	}
}

func TestPurgedURLForgetsItsVersion(t *testing.T) {
	h := openHarness(t)
	indexed := h.today() - 3
	h.admit(t, firstOrigin, postingOf("alpha", indexed), indexed)

	h.purgeURL(t, urlHash("page"))

	if verdict := h.admit(
		t,
		secondOrigin,
		postingOf("beta", indexed-1),
		indexed-1,
	); verdict != urlpageversion.VerdictNewURL {
		t.Fatalf("verdict after purge = %q, want a first version", verdict)
	}
}

func TestEachVerdictNamesWhatHappenedToThePosting(t *testing.T) {
	for verdict, want := range map[urlpageversion.Verdict]urlpageversion.Outcome{
		urlpageversion.VerdictNewURL:            urlpageversion.OutcomeIndexed,
		urlpageversion.VerdictMissingWord:       urlpageversion.OutcomeIndexed,
		urlpageversion.VerdictNewerVersion:      urlpageversion.OutcomeIndexed,
		urlpageversion.VerdictNewerThanMetadata: urlpageversion.OutcomeHeld,
		urlpageversion.VerdictFutureDate:        urlpageversion.OutcomeDropped,
		urlpageversion.VerdictOlderVersion:      urlpageversion.OutcomeDropped,
		urlpageversion.VerdictSameVersion:       urlpageversion.OutcomeDropped,
		urlpageversion.VerdictSwapLimit:         urlpageversion.OutcomeDropped,
	} {
		if got := verdict.Outcome(); got != want {
			t.Errorf("%s outcome = %s, want %s", verdict, got, want)
		}
	}
}
