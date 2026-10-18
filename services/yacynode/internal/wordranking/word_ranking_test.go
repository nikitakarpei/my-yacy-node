package wordranking_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/wordranking"
)

func TestTheWordsOfARankingKeepTheirOrder(t *testing.T) {
	want := []yacymodel.Hash{
		yacymodel.WordHash("the"),
		yacymodel.WordHash("berlin"),
		yacymodel.WordHash("zebra"),
	}

	words, err := wordranking.WordsFrom(rankingFileOf(t, want[0].String()+"\n"+
		want[1].String()+"\n"+want[2].String()+"\n"))
	if err != nil {
		t.Fatalf("WordsFrom: %v", err)
	}

	if !slices.Equal(words, want) {
		t.Fatalf("words = %v, want %v", words, want)
	}
}

func TestNoRankingFileRanksNoWords(t *testing.T) {
	words, err := wordranking.WordsFrom("")
	if err != nil {
		t.Fatalf("WordsFrom: %v", err)
	}

	if len(words) != 0 {
		t.Fatalf("words = %v, want none", words)
	}
}

func TestAMissingRankingFileIsRefused(t *testing.T) {
	_, err := wordranking.WordsFrom(filepath.Join(t.TempDir(), "absent"))

	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("WordsFrom = %v, want %v", err, fs.ErrNotExist)
	}
}

func TestALineThatIsNoWordHashIsRefused(t *testing.T) {
	ranking := yacymodel.WordHash("the").String() + "\n" + "the\n"

	_, err := wordranking.WordsFrom(rankingFileOf(t, ranking))

	if !errors.Is(err, yacymodel.ErrInvalidHash) {
		t.Fatalf("WordsFrom = %v, want %v", err, yacymodel.ErrInvalidHash)
	}
}

func rankingFileOf(t *testing.T, ranking string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "word-ranking.txt")
	if err := os.WriteFile(path, []byte(ranking), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	return path
}
