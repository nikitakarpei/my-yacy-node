package yacynodedoc_test

import (
	"encoding/json"
	"testing"

	yacynodedoc "github.com/nikitakarpei/yacy-rwi-node/yacynode/doc"
)

func TestGrafanaDashboardNamesTheService(t *testing.T) {
	var dashboard struct {
		UID string `json:"uid"`
	}
	if err := json.Unmarshal(yacynodedoc.GrafanaDashboard, &dashboard); err != nil {
		t.Fatalf("dashboard: %v", err)
	}
	if dashboard.UID != "yacynode" {
		t.Errorf("uid = %q, want yacynode", dashboard.UID)
	}
}
