package yacymodel_test

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"maps"
	"math/rand/v2"
	"reflect"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

var partitionExponents = []uint{0, 1, 4, 16, 62}

func TestRootIgnoresPostingOrder(t *testing.T) {
	random := seededRandom(1)
	for _, exponent := range partitionExponents {
		partitions := partitionsFrom(t, exponent)
		postings := randomPagePostings(random, 200)
		root := rootOf(t, wholePagePostingsOf(t, postings, partitions))

		shuffled := slices.Clone(postings)
		random.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		if got := rootOf(t, wholePagePostingsOf(t, shuffled, partitions)); got != root {
			t.Errorf(
				"exponent %d: shuffled root = %x, want %x",
				exponent,
				got.Bytes(),
				root.Bytes(),
			)
		}
	}
}

func seededRandom(seed uint64) *rand.Rand {
	//nolint:gosec // G404: a seeded generator makes the random postings repeatable.
	return rand.New(rand.NewPCG(seed, seed))
}

func partitionsFrom(t *testing.T, exponent uint) yacymodel.DHTRingPartitions {
	t.Helper()
	partitions, err := yacymodel.DHTRingPartitionsFromExponent(exponent)
	if err != nil {
		t.Fatal(err)
	}

	return partitions
}

func randomPagePostings(random *rand.Rand, size int) []yacymodel.RWIPosting {
	url, _ := yacymodel.ParseURLHashBytes(randomBytes(random, yacymodel.HashByteLength))
	language, _ := yacymodel.ParseLanguage("en")
	postingsByWord := map[yacymodel.Hash]yacymodel.RWIPosting{}
	for len(postingsByWord) < size {
		word, _ := yacymodel.ParseHashBytes(randomBytes(random, yacymodel.HashByteLength))
		fields := randomBytes(random, 16)
		postingsByWord[word] = yacymodel.RWIPosting{
			WordHash:               word,
			URLHash:                url,
			LastModified:           yacymodel.MicroDate(binary.BigEndian.Uint16(fields)),
			Language:               language,
			TitleWords:             fields[2],
			TextWords:              binary.BigEndian.Uint16(fields[3:]),
			Phrases:                binary.BigEndian.Uint16(fields[5:]),
			LocalLinks:             fields[7],
			ExternalLinks:          fields[8],
			URLLength:              fields[9],
			URLComponents:          fields[10],
			Appearance:             yacymodel.Appearance{IndexOf: fields[11]%2 == 0},
			TextPosition:           binary.BigEndian.Uint16(fields[12:]),
			Hits:                   fields[14],
			PhraseRelativePosition: fields[15],
		}
	}

	return slices.Collect(maps.Values(postingsByWord))
}

func randomBytes(random *rand.Rand, length int) []byte {
	raw := []byte{}
	for len(raw) < length {
		raw = binary.BigEndian.AppendUint64(raw, random.Uint64())
	}

	return raw[:length]
}

func wholePagePostingsOf(
	t *testing.T,
	postings []yacymodel.RWIPosting,
	partitions yacymodel.DHTRingPartitions,
) yacymodel.PagePostings {
	t.Helper()

	return provenPagePostingsOf(t, postings, yacymodel.EmptyPostingsProof(partitions))
}

func rootOf(t *testing.T, pagePostings yacymodel.PagePostings) yacymodel.PostingsRoot {
	t.Helper()
	root, present := pagePostings.Root().Get()
	if !present {
		t.Fatalf("no root for %d postings", pagePostings.AmountOfPostings())
	}

	return root
}

func TestNarrowedPagePostingsLeadToTheRoot(t *testing.T) {
	random := seededRandom(2)
	for _, exponent := range partitionExponents {
		partitions := partitionsFrom(t, exponent)
		postings := randomPagePostings(random, 1+random.IntN(40))
		whole := wholePagePostingsOf(t, postings, partitions)
		root := rootOf(t, whole)

		for range 5 {
			narrowed := narrowedTo(t, whole, randomSubsetOf(random, postings))
			relayed := narrowedTo(t, narrowed, randomSubsetOf(random, narrowed.Postings()))
			assertRoot(t, narrowed, root)
			assertRoot(t, relayed, root)
		}
	}
}

func TestEachPagePostingTakesTheProofOfItsNarrowing(t *testing.T) {
	random := seededRandom(12)
	for _, exponent := range partitionExponents {
		partitions := partitionsFrom(t, exponent)
		postings := randomPagePostings(random, 1+random.IntN(300))
		whole := wholePagePostingsOf(t, postings, partitions)

		for range 5 {
			source := narrowedTo(t, whole, randomSubsetOf(random, postings))
			pagePostings := source.EachPagePosting()
			if len(pagePostings) != len(source.Postings()) {
				t.Fatalf("%d page postings, want %d", len(pagePostings), len(source.Postings()))
			}
			for index, pagePosting := range pagePostings {
				narrowed := narrowedTo(t, source, source.Postings()[index:index+1])
				if pagePosting.Posting() != narrowed.Postings()[0] ||
					pagePosting.Root() != whole.Root() ||
					!bytes.Equal(pagePostingProofOf(t, pagePosting).Bytes(),
						proofOf(t, narrowed).Bytes()) {
					t.Fatalf(
						"page posting = %+v, want its narrowing %+v",
						pagePosting,
						narrowed,
					)
				}
			}
		}
	}
}

func pagePostingProofOf(
	t *testing.T,
	pagePosting yacymodel.PagePosting,
) yacymodel.PostingsProof {
	t.Helper()
	proof, proven := pagePosting.Proof().Get()
	if !proven {
		t.Fatalf("no proof for posting %+v", pagePosting.Posting())
	}

	return proof
}

func narrowedTo(
	t *testing.T,
	pagePostings yacymodel.PagePostings,
	keptPostings []yacymodel.RWIPosting,
) yacymodel.PagePostings {
	t.Helper()
	narrowed, present := pagePostings.NarrowedTo(keptPostings).Get()
	if !present {
		t.Fatalf("no narrowing to %d postings", len(keptPostings))
	}

	return narrowed
}

func randomSubsetOf(random *rand.Rand, postings []yacymodel.RWIPosting) []yacymodel.RWIPosting {
	share := random.Float64()
	subset := []yacymodel.RWIPosting{postings[random.IntN(len(postings))]}
	for _, posting := range postings {
		if random.Float64() < share && posting != subset[0] {
			subset = append(subset, posting)
		}
	}

	return subset
}

func assertRoot(
	t *testing.T,
	pagePostings yacymodel.PagePostings,
	root yacymodel.PostingsRoot,
) {
	t.Helper()
	if got := rootOf(t, pagePostings); got != root {
		t.Fatalf("root of %d postings = %+v, want %+v", pagePostings.AmountOfPostings(), got, root)
	}
}

func TestMergedAndPurgedProofsKeepTheRootAndPurgedProofsAreMinimal(t *testing.T) {
	random := seededRandom(3)
	for _, exponent := range partitionExponents {
		partitions := partitionsFrom(t, exponent)
		postings := randomPagePostings(random, 2+random.IntN(16))
		whole := wholePagePostingsOf(t, postings, partitions)
		root := rootOf(t, whole)

		for range 2 {
			holders := []postingsWithProof{
				postingsWithProofOf(t, narrowedTo(t, whole, randomSubsetOf(random, postings))),
				postingsWithProofOf(t, narrowedTo(t, whole, randomSubsetOf(random, postings))),
			}
			for range 10 {
				holder := random.IntN(2)
				source := whole
				if other := holders[1-holder]; random.IntN(2) == 0 && len(other.postings) > 0 {
					source = provenPagePostingsOf(t, other.postings, other.proof)
				}
				holders[holder] = mergedOrPurged(t, random, holders[holder], source, whole)

				proven := holders[holder]
				if len(proven.postings) == 0 {
					continue
				}
				provenPagePostings := provenPagePostingsOf(t, proven.postings, proven.proof)
				assertRoot(t, narrowedTo(t, provenPagePostings, proven.postings), root)
				assertRoot(
					t,
					narrowedTo(t, provenPagePostings, randomSubsetOf(random, proven.postings)),
					root,
				)
			}
		}
	}
}

type postingsWithProof struct {
	postings []yacymodel.RWIPosting
	proof    yacymodel.PostingsProof
}

func postingsWithProofOf(t *testing.T, pagePostings yacymodel.PagePostings) postingsWithProof {
	t.Helper()

	return postingsWithProof{postings: pagePostings.Postings(), proof: proofOf(t, pagePostings)}
}

func proofOf(t *testing.T, pagePostings yacymodel.PagePostings) yacymodel.PostingsProof {
	t.Helper()
	proof, proven := pagePostings.Proof().Get()
	if !proven {
		t.Fatalf("no proof for %d postings", pagePostings.AmountOfPostings())
	}

	return proof
}

func provenPagePostingsOf(
	t *testing.T,
	postings []yacymodel.RWIPosting,
	proof yacymodel.PostingsProof,
) yacymodel.PagePostings {
	t.Helper()
	pagePostings, err := yacymodel.ProvenPagePostingsFrom(postings, proof)
	if err != nil {
		t.Fatal(err)
	}

	return pagePostings
}

func mergedOrPurged(
	t *testing.T,
	random *rand.Rand,
	proven postingsWithProof,
	source yacymodel.PagePostings,
	whole yacymodel.PagePostings,
) postingsWithProof {
	t.Helper()
	if random.IntN(2) == 0 {
		written := narrowedTo(t, source, randomSubsetOf(random, source.Postings()))
		mergedProof := proofOf(t, written)
		if len(proven.postings) > 0 {
			mergedProof = proven.proof.MergedWith(mergedProof)
		}

		return postingsWithProof{
			postings: postingsUnionOf(proven.postings, written.Postings()),
			proof:    mergedProof,
		}
	}
	if len(proven.postings) == 0 {
		return proven
	}
	purged := randomSubsetOf(random, proven.postings)
	remaining := postingsWithout(proven.postings, purged)
	if len(remaining) == 0 {
		return postingsWithProof{}
	}
	narrowedProof := proven.proof.NarrowedTo(wordsOf(remaining), purged)
	minimalProof := proofOf(t, narrowedTo(t, whole, remaining))
	if !bytes.Equal(narrowedProof.Bytes(), minimalProof.Bytes()) {
		t.Fatalf("proof after a purge = %x, want the minimal %x",
			narrowedProof.Bytes(), minimalProof.Bytes())
	}

	return postingsWithProof{postings: remaining, proof: narrowedProof}
}

func postingsUnionOf(proven, written []yacymodel.RWIPosting) []yacymodel.RWIPosting {
	union := slices.Clone(proven)
	for _, posting := range written {
		if !slices.Contains(proven, posting) {
			union = append(union, posting)
		}
	}

	return union
}

func postingsWithout(postings, purged []yacymodel.RWIPosting) []yacymodel.RWIPosting {
	return slices.DeleteFunc(slices.Clone(postings), func(posting yacymodel.RWIPosting) bool {
		return slices.Contains(purged, posting)
	})
}

func TestPartitionExponentChangesTheRoot(t *testing.T) {
	postings := randomPagePostings(seededRandom(4), 50)
	digests := map[yacymodel.PostingsRoot]uint{}
	for _, exponent := range partitionExponents {
		root := rootOf(t, wholePagePostingsOf(t, postings, partitionsFrom(t, exponent)))
		if other, taken := digests[root]; taken {
			t.Errorf("exponents %d and %d give the same root", other, exponent)
		}
		digests[root] = exponent
	}
}

func TestPostingsOfARingArcNeedAProofAlongItsTwoEdges(t *testing.T) {
	const entriesAlongTwoEdges = 2 * 72
	const largestEntryLength = 1 + 9 + 32
	random := seededRandom(5)
	for _, exponent := range partitionExponents[:4] {
		partitions := partitionsFrom(t, exponent)
		postings := randomPagePostings(random, 600)
		slices.SortFunc(postings, func(first, second yacymodel.RWIPosting) int {
			return cmp.Compare(
				yacymodel.DHTRingPositionOfPosting(first, partitions),
				yacymodel.DHTRingPositionOfPosting(second, partitions),
			)
		})

		arc := narrowedTo(t, wholePagePostingsOf(t, postings, partitions), postings[150:450])

		if length := len(
			proofOf(t, arc).Bytes(),
		); length > 2+entriesAlongTwoEdges*largestEntryLength {
			t.Errorf("exponent %d: proof of a ring arc takes %d bytes", exponent, length)
		}
	}
}

func TestPagePostingsRejectMalformedPostings(t *testing.T) {
	random := seededRandom(6)
	postings := randomPagePostings(random, 3)
	otherPage := randomPagePostings(random, 1)
	sameWordTwice := postings[0]
	sameWordTwice.Hits++
	withoutWord := postings[0]
	withoutWord.WordHash = yacymodel.Hash{}
	withoutURL := slices.Clone(postings)
	for index := range withoutURL {
		withoutURL[index].URLHash = yacymodel.URLHash{}
	}

	for name, malformed := range map[string][]yacymodel.RWIPosting{
		"no postings":     nil,
		"two pages":       slices.Concat(postings, otherPage),
		"same word twice": append(slices.Clone(postings), sameWordTwice),
		"no word hash":    {withoutWord},
		"no url hash":     withoutURL,
	} {
		if _, err := yacymodel.UnprovenPagePostingsFrom(malformed); !errors.Is(
			err,
			yacymodel.ErrBadPagePostings,
		) {
			t.Errorf("%s: UnprovenPagePostingsFrom = %v, want ErrBadPagePostings", name, err)
		}
		if _, err := yacymodel.ProvenPagePostingsFrom(
			malformed,
			yacymodel.PostingsProof{},
		); !errors.Is(
			err,
			yacymodel.ErrBadPagePostings,
		) {
			t.Errorf("%s: ProvenPagePostingsFrom = %v, want ErrBadPagePostings", name, err)
		}
	}
}

func TestZeroProofIsTheProofOfAWholePageInOnePartition(t *testing.T) {
	postings := randomPagePostings(seededRandom(22), 3)

	zeroProven := provenPagePostingsOf(t, postings, yacymodel.PostingsProof{})

	if got, want := zeroProven.Root(), wholePagePostingsOf(
		t,
		postings,
		yacymodel.DHTRingPartitions{},
	).Root(); got != want {
		t.Errorf("root with a zero proof = %+v, want the root in one partition %+v", got, want)
	}
}

func TestZeroValuesReadBackFromTheirBytes(t *testing.T) {
	root, err := yacymodel.ParsePostingsRootBytes(yacymodel.PostingsRoot{}.Bytes())
	if err != nil || root != (yacymodel.PostingsRoot{}) {
		t.Errorf("zero root read back as %+v, %v", root, err)
	}
	proof, err := yacymodel.ParsePostingsProofBytes(yacymodel.PostingsProof{}.Bytes())
	if err != nil || !bytes.Equal(proof.Bytes(), yacymodel.PostingsProof{}.Bytes()) {
		t.Errorf("zero proof read back as %x, %v", proof.Bytes(), err)
	}
}

func TestZeroPagePostingsHoldNothing(t *testing.T) {
	var zero yacymodel.PagePostings

	if zero.URL() != (yacymodel.URLHash{}) || zero.AmountOfPostings() != 0 ||
		zero.Root().Present() || zero.Proof().Present() {
		t.Errorf("zero page postings = url %v, %d postings, root %+v",
			zero.URL(), zero.AmountOfPostings(), zero.Root())
	}
	if zero.NarrowedTo(randomPagePostings(seededRandom(23), 1)).Present() {
		t.Error("zero page postings narrow to postings they do not hold")
	}
}

func TestPagePostingsWithoutProofHaveNoRootAndNarrowWithoutProof(t *testing.T) {
	postings := randomPagePostings(seededRandom(17), 3)
	unproven, err := yacymodel.UnprovenPagePostingsFrom(postings)
	if err != nil {
		t.Fatal(err)
	}

	if unproven.Root().Present() {
		t.Error("root is present without a proof")
	}
	if narrowedTo(t, unproven, postings[:1]).Proof().Present() {
		t.Error("narrowed proof is present without a proof")
	}
	pagePostings := unproven.EachPagePosting()
	if len(pagePostings) != len(postings) {
		t.Fatalf("%d page postings, want %d", len(pagePostings), len(postings))
	}
	for index, pagePosting := range pagePostings {
		if pagePosting.Posting() != postings[index] || pagePosting.Proof().Present() ||
			pagePosting.Root().Present() {
			t.Errorf("page posting = %+v, want posting %+v without proof or root",
				pagePosting, postings[index])
		}
	}
}

func TestNarrowedToIsAbsentForKeptPostingsOutsideThePage(t *testing.T) {
	random := seededRandom(18)
	postings := randomPagePostings(random, 3)
	whole := wholePagePostingsOf(t, postings, partitionsFrom(t, 4))
	sameWordTwice := postings[0]
	sameWordTwice.Hits++

	notHeld := randomPagePostings(random, 1)[0]
	notHeld.URLHash = postings[0].URLHash

	for name, kept := range map[string][]yacymodel.RWIPosting{
		"no postings":                 nil,
		"another page":                randomPagePostings(random, 1),
		"same word twice":             {postings[0], sameWordTwice},
		"same posting twice":          {postings[0], postings[0]},
		"same url, posting not held":  {postings[0], notHeld},
		"same word, posting not held": {sameWordTwice},
	} {
		if whole.NarrowedTo(kept).Present() {
			t.Errorf("%s: narrowing is present", name)
		}
	}
}

func TestPagePostingsRejectAProofEntryBesideNoPathOfThePostings(t *testing.T) {
	random := seededRandom(11)
	postings := randomPagePostings(random, 2)
	leafOfFirstPosting := slices.Concat(
		[]byte{1, 0, postingsTreeDepth},
		postings[0].WordHash.Bytes(),
		randomBytes(random, sha256.Size),
	)

	for name, raw := range map[string][]byte{
		"root as an entry":           slices.Concat([]byte{1, 0, 0}, randomBytes(random, sha256.Size)),
		"entry on a path":            leafOfFirstPosting,
		"many entries off the paths": slices.Concat([]byte{1, 0}, junkLeafEntries(random, 73)),
	} {
		proof, err := yacymodel.ParsePostingsProofBytes(raw)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := yacymodel.ProvenPagePostingsFrom(postings, proof); err == nil {
			t.Errorf("%s: page postings accepted the proof", name)
		}
	}
}

const postingsTreeDepth = yacymodel.HashByteLength * 8

func junkLeafEntries(random *rand.Rand, amount int) []byte {
	prefixes := make([][]byte, 0, amount)
	for range amount {
		prefixes = append(prefixes, randomBytes(random, yacymodel.HashByteLength))
	}
	slices.SortFunc(prefixes, bytes.Compare)
	prefixes = slices.CompactFunc(prefixes, bytes.Equal)
	entries := make([]byte, 0, len(prefixes)*(1+yacymodel.HashByteLength+sha256.Size))
	for _, prefix := range prefixes {
		entries = append(entries, postingsTreeDepth)
		entries = append(entries, prefix...)
		entries = append(entries, randomBytes(random, sha256.Size)...)
	}

	return entries
}

func TestPagePostingsRejectAJunkProofOfManyEntriesForOnePosting(t *testing.T) {
	random := seededRandom(16)
	posting := randomPagePostings(random, 1)
	proof, err := yacymodel.ParsePostingsProofBytes(
		slices.Concat([]byte{1, 0}, junkLeafEntries(random, 75000)),
	)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := yacymodel.ProvenPagePostingsFrom(posting, proof); err == nil {
		t.Error("page postings accepted a junk proof")
	}
}

func TestRootDiffersForForgedPagePostings(t *testing.T) {
	random := seededRandom(10)
	partitions := partitionsFrom(t, 4)
	postings := randomPagePostings(random, 40)
	root := rootOf(t, wholePagePostingsOf(t, postings, partitions))
	kept := narrowedTo(t, wholePagePostingsOf(t, postings, partitions), postings[:10])
	keptProof := proofOf(t, kept)
	stranger := randomPagePostings(random, 1)[0]

	changed := kept.Postings()
	changed[0].Hits++
	moved := kept.Postings()
	moved[0].WordHash = stranger.WordHash
	onOtherPage := kept.Postings()
	for i := range onOtherPage {
		onOtherPage[i].URLHash = stranger.URLHash
	}
	stranger.URLHash = postings[0].URLHash

	for name, forged := range map[string]struct {
		postings []yacymodel.RWIPosting
		proof    yacymodel.PostingsProof
	}{
		"changed posting beside its true leaf": {
			changed,
			keptProof.NarrowedTo(wordsOf(kept.Postings()[1:]), kept.Postings()[:1]),
		},
		"extra posting":                 {append(kept.Postings(), stranger), keptProof},
		"same words on another page":    {onOtherPage, keptProof},
		"posting moved to another word": {moved, keptProof},
	} {
		pagePostings, err := yacymodel.ProvenPagePostingsFrom(forged.postings, forged.proof)
		if err == nil && pagePostings.Root() == yacymodel.Some(root) {
			t.Errorf("%s: forged postings give the root", name)
		}
	}
}

func TestEntryOfAnEmptySubtreeKeepsTheRoot(t *testing.T) {
	random := seededRandom(14)
	postings := randomPagePostings(random, 20)
	root := rootOf(t, wholePagePostingsOf(t, postings, partitionsFrom(t, 0)))
	emptyLeaf := sha256.Sum256([]byte{2, 0})
	besideFirstLeaf := postings[0].WordHash.Bytes()
	besideFirstLeaf[len(besideFirstLeaf)-1] ^= 1
	entry, err := yacymodel.ParsePostingsProofBytes(slices.Concat(
		[]byte{1, 0, postingsTreeDepth},
		besideFirstLeaf,
		emptyLeaf[:],
	))
	if err != nil {
		t.Fatal(err)
	}

	withEntry := provenPagePostingsOf(t, postings, entry)

	if got := rootOf(t, withEntry); got != root {
		t.Errorf("root with an empty subtree entry = %x, want %x", got.Bytes(), root.Bytes())
	}
}

func TestRootChangesWithEachFieldOfAPosting(t *testing.T) {
	random := seededRandom(7)
	partitions := partitionsFrom(t, 4)
	posting := randomPagePostings(random, 1)[0]
	root := rootOf(t, wholePagePostingsOf(t, []yacymodel.RWIPosting{posting}, partitions))
	stranger := randomPagePostings(random, 1)[0]
	french, err := yacymodel.ParseLanguage("fr")
	if err != nil {
		t.Fatal(err)
	}
	replacements := map[string]any{
		"WordHash": stranger.WordHash,
		"URLHash":  stranger.URLHash,
		"Language": french,
	}

	for _, path := range fieldPathsOf(reflect.TypeFor[yacymodel.RWIPosting]()) {
		changed := posting
		field := reflect.ValueOf(&changed).Elem().FieldByIndex(path)
		name := reflect.TypeFor[yacymodel.RWIPosting]().FieldByIndex(path).Name
		switch replacement, replaced := replacements[name]; {
		case replaced:
			field.Set(reflect.ValueOf(replacement))
		case field.Kind() == reflect.Bool:
			field.SetBool(!field.Bool())
		case field.CanUint():
			field.SetUint(field.Uint() + 1)
		default:
			t.Fatalf("no change for field %s", name)
		}

		pagePostings := wholePagePostingsOf(t, []yacymodel.RWIPosting{changed}, partitions)
		if rootOf(t, pagePostings) == root {
			t.Errorf("a changed %s keeps the root", name)
		}
	}
}

func fieldPathsOf(structType reflect.Type) [][]int {
	paths := [][]int{}
	for i := range structType.NumField() {
		field := structType.Field(i)
		if field.Type != reflect.TypeFor[yacymodel.Appearance]() {
			paths = append(paths, []int{i})

			continue
		}
		for _, flagPath := range fieldPathsOf(field.Type) {
			paths = append(paths, append([]int{i}, flagPath...))
		}
	}

	return paths
}

func TestFrozenFormatsGiveFixedBytes(t *testing.T) {
	postings := goldenPagePostings(t)
	whole := wholePagePostingsOf(t, postings, partitionsFrom(t, 2))

	root := rootOf(t, whole)
	proof := proofOf(t, narrowedTo(t, whole, postings[1:2]))

	if got := hex.EncodeToString(root.Bytes()); got != goldenRootBytes {
		t.Errorf("root bytes = %s, want %s", got, goldenRootBytes)
	}
	if got := hex.EncodeToString(proof.Bytes()); got != goldenProofBytes {
		t.Errorf("proof bytes = %s, want %s", got, goldenProofBytes)
	}
}

const (
	goldenRootBytes  = "01029f2615a6221687b07dadb2d6de2ab5aaa39f9300d1ce5d0cd3c5eb2eba055426"
	goldenProofBytes = "0102024064f4828e0df331f2ac5edb1259a4ff657c394af677c265e83eaa316d16" +
		"480a0d0180adf1dc92d53625320b3e4f9f715596303d633b89599f548d2402428d4a9cab12"
)

func goldenPagePostings(t *testing.T) []yacymodel.RWIPosting {
	t.Helper()
	url := urlHashFrom(t, "AbCdEfGhIjKl")
	language, err := yacymodel.ParseLanguage("de")
	if err != nil {
		t.Fatal(err)
	}

	return []yacymodel.RWIPosting{
		{
			WordHash:               mustParseHash(t, "Zx9_-aB3cD4e"),
			URLHash:                url,
			LastModified:           20000,
			Language:               language,
			DocumentType:           yacymodel.DocumentTypeHTML,
			TitleWords:             3,
			TextWords:              400,
			Phrases:                30,
			LocalLinks:             12,
			ExternalLinks:          5,
			URLLength:              40,
			URLComponents:          4,
			Appearance:             yacymodel.Appearance{AppearsInTitle: true, Emphasized: true},
			TextPosition:           260,
			Hits:                   2,
			PhraseRelativePosition: 1,
			PhrasePosition:         7,
		},
		{WordHash: mustParseHash(t, "AAAAAAAAAAAB"), URLHash: url, Language: language},
		{WordHash: mustParseHash(t, "mNoPqRsTuVwX"), URLHash: url, Language: language, Hits: 1},
	}
}
