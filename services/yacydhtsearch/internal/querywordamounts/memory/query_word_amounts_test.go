package memory_test

import (
	"maps"
	"testing"
	"time"

	querywordamountsmemory "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/internal/querywordamounts/memory"
	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
)

const capacity = 10

func TestARememberedAmountIsReturnedForItsWordOnly(t *testing.T) {
	t.Parallel()

	amounts := querywordamountsmemory.New(capacity, time.Hour)
	berlin, weather := yacymodel.WordHash("berlin"), yacymodel.WordHash("weather")
	amounts.Remember(t.Context(), map[yacymodel.Hash]int{berlin: 12})

	got := amounts.AmountsOf(t.Context(), []yacymodel.Hash{berlin, weather})

	if want := map[yacymodel.Hash]int{berlin: 12}; !maps.Equal(got, want) {
		t.Fatalf("AmountsOf = %v, want %v", got, want)
	}
}

func TestAnAmountIsForgottenAfterItsLifetime(t *testing.T) {
	t.Parallel()

	amounts := querywordamountsmemory.New(capacity, 20*time.Millisecond)
	berlin := yacymodel.WordHash("berlin")
	amounts.Remember(t.Context(), map[yacymodel.Hash]int{berlin: 12})

	time.Sleep(200 * time.Millisecond)

	if got := amounts.AmountsOf(t.Context(), []yacymodel.Hash{berlin}); len(got) != 0 {
		t.Fatalf("AmountsOf = %v, want the amount forgotten", got)
	}
}
