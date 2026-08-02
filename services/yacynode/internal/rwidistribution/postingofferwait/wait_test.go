package postingofferwait

import (
	"context"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/memvault"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/vault"
)

var testBounds = Bounds{First: time.Minute, Longest: 8 * time.Minute}

func openWait(t *testing.T) (*vault.Vault, *Wait) {
	t.Helper()

	v, err := memvault.Open(0)
	if err != nil {
		t.Fatalf("memvault.Open: %v", err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	waits, err := Open(v)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	return v, waits
}

func widen(t *testing.T, v *vault.Vault, waits *Wait) time.Duration {
	t.Helper()

	var widened time.Duration
	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		var err error
		widened, err = waits.Widen(tx, testWord, testURL, testBounds)

		return err
	}); err != nil {
		t.Fatalf("Widen: %v", err)
	}

	return widened
}

func forget(t *testing.T, v *vault.Vault, waits *Wait) {
	t.Helper()

	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return waits.Forget(tx, testWord, testURL)
	}); err != nil {
		t.Fatalf("Forget: %v", err)
	}
}

var testWord = yacymodel.WordHash("w1")

var testURL = func() yacymodel.URLHash {
	hash, err := yacymodel.ParseURLHash(yacymodel.WordHash("u1").String())
	if err != nil {
		panic(err)
	}

	return hash
}()

func TestFirstMissWaitsTheFirstBound(t *testing.T) {
	v, waits := openWait(t)

	if widened := widen(t, v, waits); widened != testBounds.First {
		t.Fatalf("Widen = %v, want %v on the first miss", widened, testBounds.First)
	}
}

func TestFurtherMissesDoubleTheWaitUpToTheLongestBound(t *testing.T) {
	v, waits := openWait(t)

	for _, want := range []time.Duration{
		time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 8 * time.Minute,
	} {
		if widened := widen(t, v, waits); widened != want {
			t.Fatalf("Widen = %v, want %v", widened, want)
		}
	}
}

func TestForgetReturnsThePostingToTheFirstBound(t *testing.T) {
	v, waits := openWait(t)

	widen(t, v, waits)
	widen(t, v, waits)
	forget(t, v, waits)

	if widened := widen(t, v, waits); widened != testBounds.First {
		t.Fatalf("Widen = %v, want %v after Forget", widened, testBounds.First)
	}
}

func TestPostingPurgedReturnsThePostingToTheFirstBound(t *testing.T) {
	v, waits := openWait(t)

	widen(t, v, waits)
	if err := v.Update(context.Background(), func(tx *vault.Txn) error {
		return waits.PostingPurged(tx, testWord, testURL)
	}); err != nil {
		t.Fatalf("PostingPurged: %v", err)
	}

	if widened := widen(t, v, waits); widened != testBounds.First {
		t.Fatalf("Widen = %v, want %v after the posting was purged", widened, testBounds.First)
	}
}
