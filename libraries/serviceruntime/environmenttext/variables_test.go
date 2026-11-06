package environmenttext_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/environmenttext"
)

func TestVariablesFromReadsEachAssignment(t *testing.T) {
	variables, err := environmenttext.VariablesFrom(
		"# broker\nSCRAPE_NATS_URL = nats://10.0.0.2:4222\n\nSCRAPE_USER_AGENT=pagescrape (+https://example.org/?site=a)\n",
	)
	if err != nil {
		t.Fatalf("VariablesFrom: %v", err)
	}
	if got := variables.Get("SCRAPE_NATS_URL"); got != "nats://10.0.0.2:4222" {
		t.Errorf("SCRAPE_NATS_URL = %q", got)
	}
	if got := variables.Get(
		"SCRAPE_USER_AGENT",
	); got != "pagescrape (+https://example.org/?site=a)" {
		t.Errorf("SCRAPE_USER_AGENT = %q, want the value with its = sign", got)
	}
	if got := variables.Get("LOG_LEVEL"); got != "" {
		t.Errorf("LOG_LEVEL = %q, want empty for a variable the text does not set", got)
	}
}

func TestVariablesFromRefusesALineWithoutAnAssignment(t *testing.T) {
	if _, err := environmenttext.VariablesFrom("SCRAPE_NATS_URL"); err == nil {
		t.Fatal("want an error for a line without =")
	}
}

func TestVariablesFromRefusesAnAssignmentWithoutAName(t *testing.T) {
	if _, err := environmenttext.VariablesFrom("=nats://10.0.0.2:4222"); err == nil {
		t.Fatal("want an error for a line without a name")
	}
}

func TestVariablesFromRefusesAVariableSetTwice(t *testing.T) {
	if _, err := environmenttext.VariablesFrom("LOG_LEVEL=debug\nLOG_LEVEL=info"); err == nil {
		t.Fatal("want an error for a variable set twice")
	}
}
