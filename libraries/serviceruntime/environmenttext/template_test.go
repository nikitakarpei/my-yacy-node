package environmenttext_test

import (
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/environmenttext"
)

func TestTemplateOfListsEachVariableWithItsDefault(t *testing.T) {
	template := environmenttext.TemplateOf([]environmenttext.Variable{
		{Name: "SCRAPE_NATS_URL"},
		{Name: "SCRAPE_PROXY_DIAL_MODE", Default: "tunnel"},
	})

	if template != "SCRAPE_NATS_URL=\nSCRAPE_PROXY_DIAL_MODE=tunnel\n" {
		t.Errorf("template = %q", template)
	}
}

func TestTemplateOfReadsBackAsTheDefaults(t *testing.T) {
	variables, err := environmenttext.VariablesFrom(
		environmenttext.TemplateOf([]environmenttext.Variable{
			{Name: "SCRAPE_PROXY_DIAL_MODE", Default: "tunnel"},
		}),
	)
	if err != nil {
		t.Fatalf("VariablesFrom: %v", err)
	}
	if got := variables.Get("SCRAPE_PROXY_DIAL_MODE"); got != "tunnel" {
		t.Errorf("SCRAPE_PROXY_DIAL_MODE = %q, want the default", got)
	}
}
