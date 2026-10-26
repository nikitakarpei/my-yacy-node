package yacycrawlerdoc_test

import (
	"encoding/json"
	"testing"

	yacycrawlerdoc "github.com/nikitakarpei/yacy-rwi-node/yacycrawler/doc"
)

func TestGrafanaDashboardNamesTheService(t *testing.T) {
	var dashboard struct {
		UID string `json:"uid"`
	}
	if err := json.Unmarshal(yacycrawlerdoc.GrafanaDashboard, &dashboard); err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if dashboard.UID != "yacycrawler" {
		t.Errorf("uid = %q, want yacycrawler", dashboard.UID)
	}
}
