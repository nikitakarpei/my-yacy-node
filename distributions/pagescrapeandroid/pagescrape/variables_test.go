package pagescrape_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	pagescrapeassembly "github.com/nikitakarpei/yacy-rwi-node/pagescrape/assembly"
	"github.com/nikitakarpei/yacy-rwi-node/pagescrapeandroid/pagescrape"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/applog"
	"github.com/nikitakarpei/yacy-rwi-node/serviceruntime/environmenttext"
)

var requiredValues = map[string]string{
	pagescrapeassembly.EnvScrapeNATSURL: "nats://127.0.0.1:4222",
	pagescrapeassembly.EnvProxyURL:      "http://127.0.0.1:8787",
}

func TestVariablesListEachVariableTheServiceReads(t *testing.T) {
	var readNames []string
	getenv := func(name string) string {
		readNames = append(readNames, name)
		return requiredValues[name]
	}
	if _, err := pagescrapeassembly.LoadServiceConfig(getenv); err != nil {
		t.Fatalf("LoadServiceConfig: %v", err)
	}
	if err := applog.Configure(getenv); err != nil {
		t.Fatalf("Configure: %v", err)
	}

	listedNames := namesIn(pagescrape.Variables())
	slices.Sort(readNames)
	if !slices.Equal(slices.Compact(readNames), listedNames) {
		t.Errorf("listed %v, want the variables the service reads, %v", listedNames, readNames)
	}
}

func TestVariablesGiveTheDefaultsOfTheService(t *testing.T) {
	defaults, err := environmenttext.VariablesFrom(pagescrape.Variables())
	if err != nil {
		t.Fatalf("VariablesFrom: %v", err)
	}
	withDefaults := func(name string) string {
		if value, required := requiredValues[name]; required {
			return value
		}
		return defaults.Get(name)
	}
	withoutValues := func(name string) string { return requiredValues[name] }

	fromDefaults, err := pagescrapeassembly.LoadServiceConfig(withDefaults)
	if err != nil {
		t.Fatalf("LoadServiceConfig with the listed defaults: %v", err)
	}
	fromNothing, err := pagescrapeassembly.LoadServiceConfig(withoutValues)
	if err != nil {
		t.Fatalf("LoadServiceConfig without values: %v", err)
	}
	if !reflect.DeepEqual(fromDefaults, fromNothing) {
		t.Errorf(
			"config from the listed defaults = %+v, want the defaults of the service %+v",
			fromDefaults,
			fromNothing,
		)
	}
}

func namesIn(template string) []string {
	var names []string
	for line := range strings.Lines(template) {
		name, _, _ := strings.Cut(line, "=")
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}
