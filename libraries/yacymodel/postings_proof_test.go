package yacymodel_test

import (
	"bytes"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

func TestProofBytesRoundTrip(t *testing.T) {
	postings := randomPagePostings(seededRandom(9), 100)
	whole := wholePagePostingsOf(t, postings, partitionsFrom(t, 5))
	purgedProof := proofOf(t, narrowedTo(t, whole, postings[:30]))
	purgedProof = purgedProof.NarrowedTo(wordsOf(postings[10:30]), postings[:10])

	parsed, err := yacymodel.ParsePostingsProofBytes(purgedProof.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parsed.Bytes(), purgedProof.Bytes()) {
		t.Errorf("parsed proof = %x, want %x", parsed.Bytes(), purgedProof.Bytes())
	}
	reproven := provenPagePostingsOf(t, postings[10:30], parsed)
	if got := rootOf(t, reproven); got != rootOf(t, whole) {
		t.Errorf("root from parsed proof = %x, want %x", got.Bytes(), rootOf(t, whole).Bytes())
	}
}

func TestProofNarrowedPastARepeatedPostingRoundTrips(t *testing.T) {
	postings := randomPagePostings(seededRandom(12), 3)
	proof := proofOf(t, wholePagePostingsOf(t, postings, partitionsFrom(t, 4)))
	narrowed := proof.NarrowedTo(
		wordsOf(postings[1:]),
		[]yacymodel.RWIPosting{postings[0], postings[0]},
	)

	parsed, err := yacymodel.ParsePostingsProofBytes(narrowed.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parsed.Bytes(), narrowed.Bytes()) {
		t.Errorf("parsed proof = %x, want %x", parsed.Bytes(), narrowed.Bytes())
	}
}

func wordsOf(postings []yacymodel.RWIPosting) []yacymodel.Hash {
	words := make([]yacymodel.Hash, 0, len(postings))
	for _, posting := range postings {
		words = append(words, posting.WordHash)
	}

	return words
}

func TestProofPurgedWordByWordStaysMinimal(t *testing.T) {
	postings := randomPagePostings(seededRandom(15), 300)
	whole := wholePagePostingsOf(t, postings, partitionsFrom(t, 4))
	purgedProof := proofOf(t, whole)

	for purged := range 297 {
		remaining := postings[purged+1:]
		purgedProof = purgedProof.NarrowedTo(wordsOf(remaining), postings[purged:purged+1])

		minimalProof := proofOf(t, narrowedTo(t, whole, remaining))
		if !bytes.Equal(purgedProof.Bytes(), minimalProof.Bytes()) {
			t.Fatalf("after %d purges the proof takes %d bytes, want the minimal %d",
				purged+1, len(purgedProof.Bytes()), len(minimalProof.Bytes()))
		}
	}
}

func TestParsePostingsProofBytesRejectsMalformedBytes(t *testing.T) {
	hash := make([]byte, 32)
	entryAt := func(depth byte, prefix ...byte) []byte {
		return append(append([]byte{depth}, prefix...), hash...)
	}
	header := []byte{1, 4}

	for name, raw := range map[string][]byte{
		"empty":                 {},
		"other format":          {2, 4},
		"exponent out of range": {1, 63},
		"too deep":              slices.Concat(header, entryAt(73, make([]byte, 10)...)),
		"truncated prefix":      slices.Concat(header, []byte{72, 0xAB}),
		"truncated hash":        slices.Concat(header, entryAt(8, 0xAB)[:20]),
		"bits past the depth":   slices.Concat(header, entryAt(3, 0xFF)),
		"same entry twice":      slices.Concat(header, entryAt(2, 0x40), entryAt(2, 0x40)),
		"entries out of order":  slices.Concat(header, entryAt(2, 0x80), entryAt(2, 0x40)),
	} {
		if _, err := yacymodel.ParsePostingsProofBytes(raw); err == nil {
			t.Errorf("%s: parse accepted %x", name, raw)
		}
	}
}
