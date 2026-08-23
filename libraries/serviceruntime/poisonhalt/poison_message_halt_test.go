package poisonhalt_test

import (
	"context"
	"errors"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/poisonhalt"
)

func TestHaltReturnsPoisonSentinelWrappingTheCause(t *testing.T) {
	cause := errors.New("bad json")

	err := poisonhalt.Halt(
		context.Background(), "yacy.crawl.page.markdown YACY_CRAWL_PAGE_MARKDOWN/7", cause,
	)

	if !errors.Is(err, poisonhalt.ErrPoisonMessage) {
		t.Fatalf("Halt error = %v, want poison sentinel", err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("Halt error = %v, want wrapped cause", err)
	}
}
