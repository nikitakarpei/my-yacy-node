package yacydhtsearchdoc_test

import (
	"encoding/json"
	"testing"

	yacydhtsearchdoc "github.com/nikitakarpei/yacy-rwi-node/yacydhtsearch/doc"
)

func TestGrafanaDashboardNamesTheService(t *testing.T) {
	var dashboard struct {
		UID string `json:"uid"`
	}
	if err := json.Unmarshal(yacydhtsearchdoc.GrafanaDashboard, &dashboard); err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if dashboard.UID != "yacydhtsearch" {
		t.Errorf("uid = %q, want yacydhtsearch", dashboard.UID)
	}
}
