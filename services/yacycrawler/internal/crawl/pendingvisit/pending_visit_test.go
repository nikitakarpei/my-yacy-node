package pendingvisit_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/canonicalurl/canonicalurltest"
	"github.com/nikitakarpei/yacy-rwi-node/yacycrawler/internal/crawl/pendingvisit"
)

func TestPendingVisitSurvivesTheFrontierStream(t *testing.T) {
	visit := pendingvisit.PendingVisit{
		OrderID: "o1",
		URL:     canonicalurltest.CanonicalURLOf(t, "http://host/page"),
		Depth:   3,
	}

	data, err := pendingvisit.MarshalPendingVisit(visit)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	read, err := pendingvisit.UnmarshalPendingVisit(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if read != visit {
		t.Fatalf("read %+v, want %+v", read, visit)
	}
}

func TestUnmarshalPendingVisitRejectsRubbish(t *testing.T) {
	if _, err := pendingvisit.UnmarshalPendingVisit([]byte("{")); err == nil {
		t.Fatal("a truncated message should not decode")
	}
}
