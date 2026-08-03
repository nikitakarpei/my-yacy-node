package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
)

func TestDHTRingTracksAcceptingPeersPerSector(t *testing.T) {
	observer := NewDHTRingMetrics(prometheus.NewRegistry())

	observer.ObservePeersAcceptingRemoteIndexPerDHTRingSector([]int{0, 2, 0})

	if got := testutil.ToFloat64(
		observer.peersAcceptingRemoteIndexPerSector.WithLabelValues("01"),
	); got != 2 {
		t.Errorf("accepting peers in sector 01 = %v, want 2", got)
	}
	if got := testutil.ToFloat64(
		observer.peersAcceptingRemoteIndexPerSector.WithLabelValues("00"),
	); got != 0 {
		t.Errorf("accepting peers in sector 00 = %v, want 0", got)
	}
	if got := testutil.CollectAndCount(observer.peersAcceptingRemoteIndexPerSector); got != 3 {
		t.Errorf("sectors = %v, want every sector reported including the empty ones", got)
	}
}

func TestDHTRingTracksReplicaRingFractions(t *testing.T) {
	observer := NewDHTRingMetrics(prometheus.NewRegistry())

	observer.ObserveReplicaRingFractions([]float64{0.1, 0.4})

	var collected dto.Metric
	if err := observer.ringFractionFromPostingToHolder.Write(&collected); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got := collected.GetHistogram().GetSampleCount(); got != 2 {
		t.Errorf("samples = %v, want one per reported fraction", got)
	}
	if got := collected.GetHistogram().GetSampleSum(); got < 0.49 || got > 0.51 {
		t.Errorf("sample sum = %v, want about 0.5", got)
	}
}
