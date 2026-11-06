package servicehealth_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/servicehealth"
)

func TestHealthIsUnhealthyBeforeTheServiceStarted(t *testing.T) {
	var health servicehealth.Health
	if health.Started() || health.Healthy() {
		t.Fatal("healthy before MarkStarted")
	}
}

func TestHealthIsHealthyOnceTheServiceStarted(t *testing.T) {
	var health servicehealth.Health
	health.MarkStarted()
	if !health.Started() || !health.Healthy() {
		t.Fatal("not healthy after MarkStarted")
	}
}

func TestHealthIsUnhealthyWhileAConnectionIsLost(t *testing.T) {
	var health servicehealth.Health
	health.MarkStarted()
	health.ConnectionLost()
	health.ConnectionLost()
	health.ConnectionRestored()
	if health.Connected() || health.Healthy() {
		t.Fatal("healthy while one of two lost connections is still down")
	}
	health.ConnectionRestored()
	if !health.Healthy() {
		t.Fatal("not healthy after every connection returned")
	}
}

func TestHealthReportsEachChange(t *testing.T) {
	changes := 0
	health := servicehealth.ReportingTo(func() { changes++ })
	health.MarkStarted()
	health.ConnectionLost()
	health.ConnectionRestored()
	if changes != 3 {
		t.Fatalf("changes = %d, want 3", changes)
	}
}
