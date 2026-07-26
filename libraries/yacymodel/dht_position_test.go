package yacymodel

import "testing"

func TestWordPosition(t *testing.T) {
	low, err := WordPosition(mustParseHash(t, "AAAAAAAAAAAA"))
	if err != nil {
		t.Fatal(err)
	}
	high, err := WordPosition(mustParseHash(t, "__________AA"))
	if err != nil {
		t.Fatal(err)
	}
	if low >= high {
		t.Errorf("expected ring order low(%d) < high(%d)", low, high)
	}
	if high != MaxDHTPosition {
		t.Errorf("WordPosition of all-last folded symbols = %d, want %d", high, MaxDHTPosition)
	}
}

func TestPositionHashRoundTrip(t *testing.T) {
	// PositionHash can only recover the first 10 symbols a position was
	// derived from; the trailing two are folded away by cardinal.
	word := mustParseHash(t, "hHJBztzcFn__")
	pos, err := WordPosition(word)
	if err != nil {
		t.Fatal(err)
	}
	if got := PositionHash(pos); got != word {
		t.Errorf("PositionHash(WordPosition(%v)) = %v, want %v", word, got, word)
	}
}

func TestPostingPosition(t *testing.T) {
	word := mustParseHash(t, "hHJBztzcFn76")
	url1, err := ParseURLHash("AAAAAAAAAAAA")
	if err != nil {
		t.Fatal(err)
	}
	url2, err := ParseURLHash("____________")
	if err != nil {
		t.Fatal(err)
	}
	partitions, err := DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatal(err)
	}

	pos1, err := PostingPosition(word, url1, partitions)
	if err != nil {
		t.Fatal(err)
	}
	pos2, err := PostingPosition(word, url2, partitions)
	if err != nil {
		t.Fatal(err)
	}
	if pos1 == pos2 {
		t.Errorf(
			"PostingPosition for the same word but different urls must differ: %d == %d",
			pos1,
			pos2,
		)
	}

	wordPos, err := WordPosition(word)
	if err != nil {
		t.Fatal(err)
	}
	shift := partitions.shiftLength()
	mask := uint64(1)<<shift - 1
	if uint64(pos1)&mask != uint64(wordPos)&mask {
		t.Errorf("PostingPosition must keep the word hash's low %d bits", shift)
	}
}

func TestDHTRingPartitionsFromExponent(t *testing.T) {
	p, err := DHTRingPartitionsFromExponent(4)
	if err != nil {
		t.Fatal(err)
	}
	if p != 16 {
		t.Errorf("DHTRingPartitionsFromExponent(4) = %d, want 16", p)
	}
	if _, err := DHTRingPartitionsFromExponent(63); err == nil {
		t.Errorf("DHTRingPartitionsFromExponent(63) should fail")
	}
}

func TestDistance(t *testing.T) {
	if d := Distance(10, 40); d != 30 {
		t.Errorf("Distance(10,40) = %d, want 30", d)
	}
	if d := Distance(40, 10); d != (MaxDHTPosition-40)+10+1 {
		t.Errorf("Distance wrap = %d", d)
	}
	if d := Distance(5, 5); d != 0 {
		t.Errorf("Distance(5,5) = %d, want 0", d)
	}
}
