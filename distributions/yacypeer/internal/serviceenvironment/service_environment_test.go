package serviceenvironment_test

import (
	"strings"
	"testing"

	"github.com/nikitakarpei/yacy-rwi-node/yacypeer/internal/serviceenvironment"
)

func processFrom(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func environmentFor(
	t *testing.T,
	wiring serviceenvironment.Wiring,
	process map[string]string,
) serviceenvironment.Environment {
	t.Helper()
	environment, err := serviceenvironment.For("corpustext", wiring, nil, nil, processFrom(process))
	if err != nil {
		t.Fatalf("For: %v", err)
	}
	return environment
}

func TestLookupPrefersTheWiredValue(t *testing.T) {
	environment := environmentFor(t, serviceenvironment.Wiring{"NATS_URL": "nats://wired"}, nil)

	if got := environment.Lookup("NATS_URL"); got != "nats://wired" {
		t.Errorf("Lookup = %q, want the wired value", got)
	}
}

func TestLookupPrefersTheServiceValueOverTheSharedValue(t *testing.T) {
	environment := environmentFor(t, nil, map[string]string{
		"CORPUSTEXT__LANGUAGES": "de",
		"LANGUAGES":             "en",
	})

	if got := environment.Lookup("LANGUAGES"); got != "de" {
		t.Errorf("Lookup = %q, want the service value", got)
	}
}

func TestLookupFallsBackToTheSharedValue(t *testing.T) {
	environment := environmentFor(t, nil, map[string]string{
		"LANGUAGES":             "en",
		"CORPUSMARKDOWN__LIMIT": "9",
	})

	if got := environment.Lookup("LANGUAGES"); got != "en" {
		t.Errorf("Lookup = %q, want the shared value", got)
	}
	if got := environment.Lookup("LIMIT"); got != "" {
		t.Errorf("Lookup = %q, want no value from another service", got)
	}
}

func TestLookupFallsBackToThePeerDefault(t *testing.T) {
	environment, err := serviceenvironment.For(
		"corpustext",
		nil,
		serviceenvironment.Defaults{"LISTEN_ADDR": "127.0.0.1:8096"},
		nil,
		processFrom(nil),
	)
	if err != nil {
		t.Fatalf("For: %v", err)
	}

	if got := environment.Lookup("LISTEN_ADDR"); got != "127.0.0.1:8096" {
		t.Errorf("Lookup = %q, want the peer default", got)
	}
}

func TestLookupPrefersTheOperatorValueOverThePeerDefault(t *testing.T) {
	environment, err := serviceenvironment.For(
		"corpustext",
		nil,
		serviceenvironment.Defaults{"LISTEN_ADDR": "127.0.0.1:8096"},
		nil,
		processFrom(map[string]string{"LISTEN_ADDR": ":8096"}),
	)
	if err != nil {
		t.Fatalf("For: %v", err)
	}

	if got := environment.Lookup("LISTEN_ADDR"); got != ":8096" {
		t.Errorf("Lookup = %q, want the operator value", got)
	}
}

func TestForRefusesEveryWiredSettingTheOperatorSets(t *testing.T) {
	_, err := serviceenvironment.For(
		"corpustext",
		serviceenvironment.Wiring{"NATS_URL": "nats://wired", "DATA_DIR": "/data"},
		nil,
		nil,
		processFrom(map[string]string{
			"NATS_URL":             "nats://elsewhere",
			"CORPUSTEXT__DATA_DIR": "/elsewhere",
		}),
	)
	if err == nil {
		t.Fatal("For accepted overridden wiring, want an error")
	}
	for _, name := range []string{"NATS_URL", "CORPUSTEXT__DATA_DIR"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error %q does not name %s", err, name)
		}
	}
}

func TestForRefusesEveryUnreadSettingTheOperatorSets(t *testing.T) {
	_, err := serviceenvironment.For(
		"corpustext",
		nil,
		nil,
		[]string{"OPS_ADDR"},
		processFrom(map[string]string{"CORPUSTEXT__OPS_ADDR": ":9999"}),
	)
	if err == nil || !strings.Contains(err.Error(), "CORPUSTEXT__OPS_ADDR") {
		t.Errorf("For = %v, want the unread setting named", err)
	}
}
