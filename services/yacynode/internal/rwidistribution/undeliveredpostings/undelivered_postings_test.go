package undeliveredpostings_test

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
	"github.com/nikitakarpei/yacy-rwi-node/wallclock"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwidistribution/undeliveredpostings"
)

const (
	partitionExponent = 4
	amountOfWords     = 4
	amountOfDocuments = 64
)

type postingIdentity struct {
	word yacymodel.Hash
	url  yacymodel.URLHash
}

type harness struct {
	vault       *vault.Vault
	partitions  yacymodel.DHTRingPartitions
	postings    *vault.Collection[postingIdentity, int]
	undelivered *undeliveredpostings.Postings
}

func openHarness(t *testing.T) harness {
	t.Helper()

	v, err := memoryvault.Open(0, nil, wallclock.Clock{})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(partitionExponent)
	if err != nil {
		t.Fatalf("DHTRingPartitionsFromExponent: %v", err)
	}

	undelivered, projection, err := undeliveredpostings.Open(v, partitions)
	if err != nil {
		t.Fatalf("undeliveredpostings.Open: %v", err)
	}
	postings, err := v.RegisterProjectedCollection(
		context.Background(),
		"rwi",
		postingIdentityKeyLayout,
		hitsCodec{},
		vault.ProjectedAs(postingOf, projection),
	)
	if err != nil {
		t.Fatalf("RegisterProjectedCollection: %v", err)
	}

	return harness{vault: v, partitions: partitions, postings: postings, undelivered: undelivered}
}

var postingIdentityKeyLayout = vault.PairKey(
	vault.TextKeyPartFrom(yacymodel.Hash.String, yacymodel.ParseHash),
	vault.TextKeyPartFrom(yacymodel.URLHash.String, yacymodel.ParseURLHash),
).KeyLayoutFor(
	func(identity postingIdentity) (yacymodel.Hash, yacymodel.URLHash) {
		return identity.word, identity.url
	},
	func(word yacymodel.Hash, url yacymodel.URLHash) postingIdentity {
		return postingIdentity{word: word, url: url}
	},
)

type hitsCodec struct{}

func (hitsCodec) Encode(hits int) ([]byte, error) {
	return []byte(strconv.Itoa(hits)), nil
}

func (hitsCodec) Decode(stored []byte) (int, error) {
	hits, err := strconv.Atoi(string(stored))
	if err != nil {
		return 0, fmt.Errorf("decode hits: %w", err)
	}

	return hits, nil
}

func postingOf(identity postingIdentity, hits int) yacymodel.RWIPosting {
	return yacymodel.RWIPosting{WordHash: identity.word, URLHash: identity.url, Hits: hits}
}

func (h harness) store(t *testing.T, hits int, identities ...postingIdentity) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		for _, identity := range identities {
			if _, err := h.postings.Put(tx, identity, hits); err != nil {
				return err
			}
		}

		return nil
	})
}

func (h harness) remove(t *testing.T, identity postingIdentity) {
	t.Helper()

	h.write(t, func(tx *vault.Txn) error {
		_, err := h.postings.Delete(tx, identity)

		return err
	})
}

func (h harness) write(t *testing.T, change func(tx *vault.Txn) error) {
	t.Helper()

	if err := h.vault.Update(context.Background(), change); err != nil {
		t.Fatalf("Update: %v", err)
	}
}

func (h harness) read(t *testing.T, view func(tx *vault.Txn) error) {
	t.Helper()

	if err := h.vault.View(context.Background(), view); err != nil {
		t.Fatalf("View: %v", err)
	}
}

func (h harness) readWhole(t *testing.T) []undeliveredpostings.Posting {
	t.Helper()

	var postings []undeliveredpostings.Posting
	seen := map[yacymodel.WordPartition]bool{}
	for _, identity := range everyPosting() {
		wordPartition := yacymodel.WordPartitionOfWordAndURL(
			identity.word,
			identity.url,
			h.partitions,
		)
		if seen[wordPartition] {
			continue
		}
		seen[wordPartition] = true
		h.read(t, func(tx *vault.Txn) error {
			read, err := h.undelivered.PostingsOf(tx, wordPartition)
			postings = append(postings, read...)

			return err
		})
	}

	return postings
}

func (h harness) markDelivered(
	t *testing.T,
	posting undeliveredpostings.Posting,
) bool {
	t.Helper()

	var wasMarked bool
	h.write(t, func(tx *vault.Txn) error {
		marked, err := h.undelivered.MarkDelivered(tx, posting)
		wasMarked = marked

		return err
	})

	return wasMarked
}

func (h harness) anyIn(t *testing.T, wordPartition yacymodel.WordPartition) bool {
	t.Helper()

	var found bool
	h.read(t, func(tx *vault.Txn) error {
		held, err := h.undelivered.AnyIn(tx, wordPartition)
		found = held

		return err
	})

	return found
}

func everyPosting() []postingIdentity {
	identities := make([]postingIdentity, 0, amountOfWords*amountOfDocuments)
	for word := range amountOfWords {
		for document := range amountOfDocuments {
			identities = append(identities, postingIdentity{
				word: yacymodel.WordHash("word" + strconv.Itoa(word)),
				url:  urlHashOf(document),
			})
		}
	}

	return identities
}

func urlHashOf(document int) yacymodel.URLHash {
	url, err := yacymodel.ParseURLHash(yacymodel.WordHash(strconv.Itoa(document)).String())
	if err != nil {
		panic(err)
	}

	return url
}

func identityOf(posting undeliveredpostings.Posting) postingIdentity {
	return postingIdentity{word: posting.WordPartition.Word, url: posting.URL}
}

func identitiesOf(postings []undeliveredpostings.Posting) []postingIdentity {
	identities := make([]postingIdentity, 0, len(postings))
	for _, posting := range postings {
		identities = append(identities, identityOf(posting))
	}

	return identities
}

func TestAReadVisitsEveryPostingOnce(t *testing.T) {
	h := openHarness(t)
	h.store(t, 1, everyPosting()...)

	readPostings := h.readWhole(t)

	if len(readPostings) != len(everyPosting()) {
		t.Fatalf("read %d postings, want %d", len(readPostings), len(everyPosting()))
	}
	for _, identity := range everyPosting() {
		if !slices.Contains(identitiesOf(readPostings), identity) {
			t.Fatalf("read did not visit %v", identity)
		}
	}
}

func TestADeliveredPostingLeavesTheRead(t *testing.T) {
	h := openHarness(t)
	h.store(t, 1, everyPosting()...)

	for _, posting := range h.readWhole(t) {
		if !h.markDelivered(t, posting) {
			t.Fatalf("MarkDelivered(%v) = false, want true", identityOf(posting))
		}
	}

	if left := h.readWhole(t); len(left) != 0 {
		t.Fatalf("read visits %v after delivery", identitiesOf(left))
	}
}

func TestAPostingReplacedAfterTheReadStaysUndelivered(t *testing.T) {
	h := openHarness(t)
	h.store(t, 1, everyPosting()...)
	readPosting := h.readWhole(t)[0]

	h.store(t, 2, identityOf(readPosting))

	if h.markDelivered(t, readPosting) {
		t.Fatal("MarkDelivered of a posting replaced after the read = true, want false")
	}
	if !slices.Contains(identitiesOf(h.readWhole(t)), identityOf(readPosting)) {
		t.Fatalf("replaced posting %v left the read", identityOf(readPosting))
	}
}

func TestARemovedPostingLeavesTheRead(t *testing.T) {
	h := openHarness(t)
	h.store(t, 1, everyPosting()...)
	readPosting := h.readWhole(t)[0]

	h.remove(t, identityOf(readPosting))

	if slices.Contains(identitiesOf(h.readWhole(t)), identityOf(readPosting)) {
		t.Fatalf("removed posting %v stays in the read", identityOf(readPosting))
	}
	if h.markDelivered(t, readPosting) {
		t.Fatal("MarkDelivered of a removed posting = true, want false")
	}
}

func (h harness) postingsOf(
	t *testing.T,
	wordPartition yacymodel.WordPartition,
) []postingIdentity {
	t.Helper()

	var readPostings []undeliveredpostings.Posting
	h.read(t, func(tx *vault.Txn) error {
		var err error
		readPostings, err = h.undelivered.PostingsOf(tx, wordPartition)

		return err
	})

	return identitiesOf(readPostings)
}

func TestPostingsOfAWordPartitionAreExactlyItsUndeliveredPostings(t *testing.T) {
	h := openHarness(t)
	h.store(t, 1, everyPosting()...)
	wordPartition := yacymodel.WordPartitionOfWordAndURL(
		everyPosting()[0].word,
		everyPosting()[0].url,
		h.partitions,
	)

	var want []postingIdentity
	for _, identity := range everyPosting() {
		if yacymodel.WordPartitionOfWordAndURL(identity.word, identity.url, h.partitions) ==
			wordPartition {
			want = append(want, identity)
		}
	}
	got := h.postingsOf(t, wordPartition)
	if len(got) != len(want) {
		t.Fatalf("PostingsOf(%+v) = %v, want %v", wordPartition, got, want)
	}
	for _, identity := range got {
		if !slices.Contains(want, identity) {
			t.Fatalf("PostingsOf(%+v) holds %v of another word partition", wordPartition, identity)
		}
	}

	for _, posting := range h.readWhole(t) {
		h.markDelivered(t, posting)
	}
	if left := h.postingsOf(t, wordPartition); len(left) != 0 {
		t.Fatalf("PostingsOf(%+v) after delivery = %v, want none", wordPartition, left)
	}
}

func TestAnyInSeesOnlyItsOwnWordPartition(t *testing.T) {
	h := openHarness(t)
	word := yacymodel.WordHash("word0")
	otherWord := yacymodel.WordHash("word1")
	url := urlHashOf(0)
	stored := yacymodel.WordPartitionOfWordAndURL(word, url, h.partitions)
	h.store(t, 1, postingIdentity{word: word, url: url})

	if !h.anyIn(t, stored) {
		t.Fatalf("AnyIn(%+v) = false, want true", stored)
	}
	otherPartition := yacymodel.WordPartition{
		Word:      word,
		Partition: (stored.Partition + 1) % uint(h.partitions),
	}
	if h.anyIn(t, otherPartition) {
		t.Fatalf("AnyIn(%+v) = true, want false", otherPartition)
	}
	otherWordPartition := yacymodel.WordPartition{Word: otherWord, Partition: stored.Partition}
	if h.anyIn(t, otherWordPartition) {
		t.Fatalf("AnyIn(%+v) = true, want false", otherWordPartition)
	}
}

func TestAnyInIsFalseOnceTheWordPartitionIsDelivered(t *testing.T) {
	h := openHarness(t)
	h.store(t, 1, everyPosting()...)
	readPostings := h.readWhole(t)

	for _, posting := range readPostings {
		h.markDelivered(t, posting)
	}

	if h.anyIn(t, readPostings[0].WordPartition) {
		t.Fatalf("AnyIn(%+v) = true after delivery, want false", readPostings[0].WordPartition)
	}
}

func (h harness) amountOfPostings(t *testing.T) int {
	t.Helper()

	var amountOfPostings int
	h.read(t, func(tx *vault.Txn) error {
		amount, err := h.undelivered.AmountOfPostings(tx)
		amountOfPostings = amount

		return err
	})

	return amountOfPostings
}

func TestAmountOfPostingsCountsTheUndeliveredOnes(t *testing.T) {
	h := openHarness(t)
	h.store(t, 1, everyPosting()...)
	readPostings := h.readWhole(t)[:len(everyPosting())/2]

	for _, posting := range readPostings {
		h.markDelivered(t, posting)
	}

	if got, want := h.amountOfPostings(t), len(everyPosting())-len(readPostings); got != want {
		t.Fatalf("AmountOfPostings = %d, want %d", got, want)
	}
}
