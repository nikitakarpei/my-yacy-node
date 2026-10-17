package remoteindexadmission_test

import (
	"context"
	"maps"
	"testing"
	"time"

	"github.com/nikitakarpei/yacy-rwi-node/yacymodel"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/remoteindexadmission"
	"github.com/nikitakarpei/yacy-rwi-node/yacynode/internal/rwiorigin"
)

func metadataOn(t *testing.T, seed string, day yacymodel.MicroDate) yacymodel.URLMetadata {
	t.Helper()

	metadata := metadataOfPostingsTo(t, seed)
	metadata.Modified = yacymodel.Some(yacymodel.CalendarDayOf(day.Time()))

	return metadata
}

func (h harness) receiveMetadata(
	t *testing.T,
	sender rwiorigin.Origin,
	metadata ...yacymodel.URLMetadata,
) remoteindexadmission.URLMetadataReceipt {
	t.Helper()

	receipt, err := h.urls.Receive(context.Background(), sender, metadata)
	if err != nil {
		t.Fatalf("urls.Receive: %v", err)
	}

	return receipt
}

func TestURLMetadataIntakeReportsOneOutcomePerRecord(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	stored := metadataOn(t, "u1", today-1)
	sameDay := stored
	sameDay.LocalLinks = 9

	h.receiveMetadata(t, sendingPeer, stored)
	receipt := h.receiveMetadata(t, sendingPeer,
		stored, metadataOn(t, "u1", today-2), sameDay, metadataOn(t, "u2", today+1),
	)

	want := map[remoteindexadmission.URLMetadataOutcome]int{
		remoteindexadmission.URLMetadataStoredNewURL:         1,
		remoteindexadmission.URLMetadataDroppedAlreadyStored: 1,
		remoteindexadmission.URLMetadataDroppedOtherVersion:  2,
		remoteindexadmission.URLMetadataDroppedFutureDate:    1,
	}
	if !maps.Equal(h.observed.urlMetadata, want) || receipt.KnownURLs != 3 {
		t.Fatalf("outcomes = %v, known urls = %d, want %v and 3 known urls",
			h.observed.urlMetadata, receipt.KnownURLs, want)
	}
}

func TestURLMetadataIntakeBusyAtCapacity(t *testing.T) {
	h := openHarness(t, usedBytesOfAnEmptyNode(t)+1, 100)
	h.receiveMetadata(t, sendingPeer, metadataOfPostingsTo(t, "u1"))

	if receipt := h.receiveMetadata(t, sendingPeer, metadataOfPostingsTo(t, "u2")); !receipt.Busy {
		t.Fatalf("receipt = %+v, want Busy", receipt)
	}
}

func TestALaterVersionPurgesTheIndexedPostingsOfItsURL(t *testing.T) {
	h := openHarness(t, 0, 100)
	today := harnessToday()
	indexed := postingOn("w1", today-5)
	h.receiveMetadata(t, sendingPeer, metadataOn(t, "u1", today-5))
	h.receive(t, []yacymodel.RWIPosting{indexed})

	h.receiveMetadata(t, sendingPeer, metadataOn(t, "u1", today))

	if h.indexed(t, indexed) ||
		h.observed.urlMetadata[remoteindexadmission.URLMetadataReplacedOlderVersion] != 1 {
		t.Fatalf("outcomes = %v, want the older version's postings purged", h.observed.urlMetadata)
	}
}

func TestURLVersionSwapsPerNetworkStopAtTheHourlyLimit(t *testing.T) {
	h := openHarness(t, 0, 100)
	base := harnessToday() - 5
	h.receiveMetadata(t, sendingPeer, metadataOn(t, "u1", base))

	h.receiveMetadata(t, sendingPeer, metadataOn(t, "u1", base+1))
	h.receiveMetadata(t, sendingPeer, metadataOn(t, "u1", base+2))
	if h.observed.urlMetadata[remoteindexadmission.URLMetadataReplacedOlderVersion] != 1 ||
		h.observed.urlMetadata[remoteindexadmission.URLMetadataDroppedOverSwapLimit] != 1 {
		t.Fatalf("outcomes = %v, want one swap and one over the limit", h.observed.urlMetadata)
	}

	h.receiveMetadata(t, otherPeer, metadataOn(t, "u1", base+2))
	*h.clock = h.clock.Add(time.Hour)
	h.receiveMetadata(t, sendingPeer, metadataOn(t, "u1", base+3))

	if h.observed.urlMetadata[remoteindexadmission.URLMetadataReplacedOlderVersion] != 3 {
		t.Fatalf("outcomes = %v, want swaps from another network and an hour later",
			h.observed.urlMetadata)
	}
}

func TestURLVersionSwapsInOneTransferStopAtTheHourlyLimit(t *testing.T) {
	h := openHarness(t, 0, 100)
	base := harnessToday() - 5
	h.receiveMetadata(t, sendingPeer, metadataOn(t, "u1", base), metadataOn(t, "u2", base))

	h.receiveMetadata(t, sendingPeer, metadataOn(t, "u1", base+1), metadataOn(t, "u2", base+1))

	if h.observed.urlMetadata[remoteindexadmission.URLMetadataReplacedOlderVersion] != 1 ||
		h.observed.urlMetadata[remoteindexadmission.URLMetadataDroppedOverSwapLimit] != 1 {
		t.Fatalf("outcomes = %v, want one swap and one over the limit", h.observed.urlMetadata)
	}
}
