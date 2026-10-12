package undeliveredpostings_test

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/vault"
	"github.com/nikitakarpei/yacy-rwi-node/vaultengines/memoryvault"
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
	undelivered *undeliveredpostings.UndeliveredPostings
}

func openHarness(t *testing.T) harness {
	t.Helper()

	v, err := memoryvault.Open(0, nil)
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

func (h harness) walk(
	t *testing.T,
	sector yacymodel.DHTRingSector,
	lastVisited undeliveredpostings.UndeliveredPosting,
	limit int,
) []undeliveredpostings.UndeliveredPosting {
	t.Helper()

	var visited []undeliveredpostings.UndeliveredPosting
	h.read(t, func(tx *vault.Txn) error {
		visited = nil

		return h.undelivered.PostingsIn(
			tx,
			sector,
			lastVisited,
			func(posting undeliveredpostings.UndeliveredPosting) (bool, error) {
				visited = append(visited, posting)

				return len(visited) < limit, nil
			},
		)
	})

	return visited
}

func (h harness) walkWhole(
	t *testing.T,
	sector yacymodel.DHTRingSector,
) []undeliveredpostings.UndeliveredPosting {
	t.Helper()

	return h.walk(t, sector, undeliveredpostings.UndeliveredPosting{}, len(everyPosting()))
}

func (h harness) markDelivered(
	t *testing.T,
	posting undeliveredpostings.UndeliveredPosting,
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

func (h harness) positionOf(
	posting undeliveredpostings.UndeliveredPosting,
) yacymodel.DHTRingPosition {
	return yacymodel.DHTRingPositionOfWordAndURL(
		posting.WordPartition.Word,
		posting.URL,
		h.partitions,
	)
}

func (h harness) sectorOf(identity postingIdentity) yacymodel.DHTRingSector {
	return yacymodel.DHTRingSectorOf(
		yacymodel.DHTRingPositionOfWordAndURL(identity.word, identity.url, h.partitions),
	)
}

func (h harness) busiestSector() yacymodel.DHTRingSector {
	amountPerSector := map[yacymodel.DHTRingSector]int{}
	busiest := yacymodel.DHTRingSector(0)
	for _, identity := range everyPosting() {
		sector := h.sectorOf(identity)
		amountPerSector[sector]++
		if amountPerSector[sector] > amountPerSector[busiest] {
			busiest = sector
		}
	}

	return busiest
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

func identityOf(posting undeliveredpostings.UndeliveredPosting) postingIdentity {
	return postingIdentity{word: posting.WordPartition.Word, url: posting.URL}
}

func identitiesOf(postings []undeliveredpostings.UndeliveredPosting) []postingIdentity {
	identities := make([]postingIdentity, 0, len(postings))
	for _, posting := range postings {
		identities = append(identities, identityOf(posting))
	}

	return identities
}

func TestAddedPostingsShowInTheirSectorInRingOrder(t *testing.T) {
	h := openHarness(t)
	h.store(t, 1, everyPosting()...)

	sector := h.busiestSector()
	visited := h.walkWhole(t, sector)

	var want []postingIdentity
	for _, identity := range everyPosting() {
		if h.sectorOf(identity) == sector {
			want = append(want, identity)
		}
	}
	if len(visited) != len(want) || len(want) < 2 {
		t.Fatalf("walk of sector %d visited %d postings, want %d", sector, len(visited), len(want))
	}
	for _, identity := range identitiesOf(visited) {
		if !slices.Contains(want, identity) {
			t.Fatalf("walk of sector %d visited %v, which sits in sector %d",
				sector, identity, h.sectorOf(identity))
		}
	}
	for index := 1; index < len(visited); index++ {
		if h.positionOf(visited[index]) < h.positionOf(visited[index-1]) {
			t.Fatalf("walk of sector %d goes back on the ring at %d", sector, index)
		}
	}
}

func TestEveryPostingShowsInExactlyOneSector(t *testing.T) {
	h := openHarness(t)
	h.store(t, 1, everyPosting()...)

	visited := make([]postingIdentity, 0, len(everyPosting()))
	for sector := range yacymodel.DHTRingSectorCount {
		visited = append(visited, identitiesOf(h.walkWhole(t, sector))...)
	}

	if len(visited) != len(everyPosting()) {
		t.Fatalf("walks of all sectors visited %d postings, want %d",
			len(visited), len(everyPosting()))
	}
	for _, identity := range everyPosting() {
		if !slices.Contains(visited, identity) {
			t.Fatalf("no sector walk visited %v", identity)
		}
	}
}

func TestAWalkResumesAfterTheLastVisitedPosting(t *testing.T) {
	h := openHarness(t)
	h.store(t, 1, everyPosting()...)
	sector := h.busiestSector()
	whole := h.walkWhole(t, sector)

	head := h.walk(t, sector, undeliveredpostings.UndeliveredPosting{}, len(whole)/2)
	rest := h.walk(t, sector, head[len(head)-1], len(whole))

	if got, want := identitiesOf(
		append(head, rest...),
	), identitiesOf(
		whole,
	); !slices.Equal(
		got,
		want,
	) {
		t.Fatalf("resumed walk visited %v, want %v", got, want)
	}
}

func TestADeliveredPostingLeavesItsSector(t *testing.T) {
	h := openHarness(t)
	h.store(t, 1, everyPosting()...)
	sector := h.busiestSector()

	for _, posting := range h.walkWhole(t, sector) {
		if !h.markDelivered(t, posting) {
			t.Fatalf("MarkDelivered(%v) = false, want true", identityOf(posting))
		}
	}

	if left := h.walkWhole(t, sector); len(left) != 0 {
		t.Fatalf("sector %d holds %v after delivery", sector, identitiesOf(left))
	}
}

func TestAPostingReplacedAfterTheWalkStaysUndelivered(t *testing.T) {
	h := openHarness(t)
	h.store(t, 1, everyPosting()...)
	sector := h.busiestSector()
	walked := h.walkWhole(t, sector)[0]

	h.store(t, 2, identityOf(walked))

	if h.markDelivered(t, walked) {
		t.Fatal("MarkDelivered of a posting replaced after the walk = true, want false")
	}
	if !slices.Contains(identitiesOf(h.walkWhole(t, sector)), identityOf(walked)) {
		t.Fatalf("replaced posting %v left its sector", identityOf(walked))
	}
}

func TestARemovedPostingLeavesItsSector(t *testing.T) {
	h := openHarness(t)
	h.store(t, 1, everyPosting()...)
	sector := h.busiestSector()
	walked := h.walkWhole(t, sector)[0]

	h.remove(t, identityOf(walked))

	if slices.Contains(identitiesOf(h.walkWhole(t, sector)), identityOf(walked)) {
		t.Fatalf("removed posting %v stays in its sector", identityOf(walked))
	}
	if h.markDelivered(t, walked) {
		t.Fatal("MarkDelivered of a removed posting = true, want false")
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
	sector := h.busiestSector()
	walked := h.walkWhole(t, sector)

	for _, posting := range walked {
		h.markDelivered(t, posting)
	}

	if h.anyIn(t, walked[0].WordPartition) {
		t.Fatalf("AnyIn(%+v) = true after delivery, want false", walked[0].WordPartition)
	}
}
